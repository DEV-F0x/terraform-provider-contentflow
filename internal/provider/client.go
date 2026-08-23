package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Client is a thin wrapper around a ContentFlow dashboard's /api/v1 JSON
// API (see services/dashboard/app/api_routes.py in the ContentFlow repo).
type Client struct {
	endpoint   string
	apiToken   string
	httpClient *http.Client
}

func NewClient(endpoint, apiToken string, httpClient *http.Client) *Client {
	return &Client{
		endpoint:   strings.TrimRight(endpoint, "/"),
		apiToken:   apiToken,
		httpClient: httpClient,
	}
}

// Asset mirrors Asset.to_dict() in services/dashboard/app/models.py.
type Asset struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
	ForceDownload    bool   `json:"force_download"`
	SizeBytes        int64  `json:"size_bytes"`
	SHA256           string `json:"sha256"`
	Integrity        string `json:"integrity"`
	URL              string `json:"url"`
	Tag              string `json:"tag"`
	CreatedAt        string `json:"created_at"`
	IsImage          bool   `json:"is_image"`
}

type apiError struct {
	Error string `json:"error"`
}

// assetFields holds the non-file multipart form fields shared by create and
// update requests. Zero-value ContentType/Name mean "don't override" for
// create, and ForceDownload being nil means "leave unchanged" for update.
type assetFields struct {
	Name          string
	ContentType   string
	ForceDownload *bool
}

func buildMultipart(fileName string, fileBytes []byte, fields assetFields) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	if fileBytes != nil {
		part, err := w.CreateFormFile("file", fileName)
		if err != nil {
			return nil, "", fmt.Errorf("building multipart file field: %w", err)
		}
		if _, err := part.Write(fileBytes); err != nil {
			return nil, "", fmt.Errorf("writing multipart file field: %w", err)
		}
	}
	if fields.Name != "" {
		_ = w.WriteField("name", fields.Name)
	}
	if fields.ContentType != "" {
		_ = w.WriteField("content_type", fields.ContentType)
	}
	if fields.ForceDownload != nil {
		value := "false"
		if *fields.ForceDownload {
			value = "true"
		}
		_ = w.WriteField("force_download", value)
	}
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}
	return body, w.FormDataContentType(), nil
}

func (c *Client) do(method, path, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.endpoint+path, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contentflow API request failed: %w", err)
	}
	return resp, nil
}

func decodeAsset(resp *http.Response) (*Asset, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, apiErrorFrom(resp.StatusCode, data)
	}
	var asset Asset
	if err := json.Unmarshal(data, &asset); err != nil {
		return nil, fmt.Errorf("decoding contentflow API response: %w", err)
	}
	return &asset, nil
}

func apiErrorFrom(statusCode int, data []byte) error {
	var apiErr apiError
	if err := json.Unmarshal(data, &apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("contentflow API error (%d): %s", statusCode, apiErr.Error)
	}
	return fmt.Errorf("contentflow API error (%d): %s", statusCode, string(data))
}

func (c *Client) CreateAsset(fileName string, fileBytes []byte, fields assetFields) (*Asset, error) {
	body, contentType, err := buildMultipart(fileName, fileBytes, fields)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(http.MethodPost, "/api/v1/assets", contentType, body)
	if err != nil {
		return nil, err
	}
	return decodeAsset(resp)
}

// GetAsset returns (nil, nil) if the asset no longer exists (HTTP 404).
func (c *Client) GetAsset(id string) (*Asset, error) {
	resp, err := c.do(http.MethodGet, "/api/v1/assets/"+id, "", nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, nil
	}
	return decodeAsset(resp)
}

func (c *Client) UpdateAsset(id, fileName string, fileBytes []byte, fields assetFields) (*Asset, error) {
	body, contentType, err := buildMultipart(fileName, fileBytes, fields)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(http.MethodPatch, "/api/v1/assets/"+id, contentType, body)
	if err != nil {
		return nil, err
	}
	return decodeAsset(resp)
}

// DeleteAsset treats a 404 as success -- the asset is already gone, which
// is the caller's desired end state.
func (c *Client) DeleteAsset(id string) error {
	resp, err := c.do(http.MethodDelete, "/api/v1/assets/"+id, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return apiErrorFrom(resp.StatusCode, data)
	}
	return nil
}
