package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AssetResource{}
	_ resource.ResourceWithConfigure   = &AssetResource{}
	_ resource.ResourceWithImportState = &AssetResource{}
	_ resource.ResourceWithModifyPlan  = &AssetResource{}
)

func NewAssetResource() resource.Resource {
	return &AssetResource{}
}

// AssetResource manages one asset (a served-name -> uploaded-file mapping)
// on a ContentFlow dashboard, via services/dashboard/app/api_routes.py.
type AssetResource struct {
	client *Client
}

type assetResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Source        types.String `tfsdk:"source"`
	SourceHash    types.String `tfsdk:"source_hash"`
	ContentType   types.String `tfsdk:"content_type"`
	ForceDownload types.Bool   `tfsdk:"force_download"`
	URL           types.String `tfsdk:"url"`
	SHA256        types.String `tfsdk:"sha256"`
	Integrity     types.String `tfsdk:"integrity"`
	SizeBytes     types.Int64  `tfsdk:"size_bytes"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *AssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_asset"
}

func (r *AssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A file hosted by ContentFlow and served at a stable URL.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The asset's database id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "The name the asset is served under: " +
					"GET /files/<name>. Also its object storage key.",
			},
			"source": schema.StringAttribute{
				Required:    true,
				Description: "Path to the local file to upload.",
			},
			"source_hash": schema.StringAttribute{
				Optional: true,
				Description: "Used to trigger re-upload when the file's contents " +
					"change, e.g. filesha256(\"path/to/file\"). Terraform can't " +
					"otherwise detect a content change from `source` (a path " +
					"string) alone.",
			},
			"content_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "MIME type to serve the asset with. Auto-detected " +
					"from the file extension when omitted.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"force_download": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "When true, served with Content-Disposition: " +
					"attachment instead of inline.",
			},
			"url": schema.StringAttribute{
				Computed:      true,
				Description:   "The asset's real, working URL -- API_PUBLIC_URL + /files/<name>.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sha256": schema.StringAttribute{
				Computed:      true,
				Description:   "SHA-256 of the uploaded file contents, hex-encoded.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"integrity": schema.StringAttribute{
				Computed:      true,
				Description:   "Subresource Integrity string (\"sha384-...\") for the asset.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"size_bytes": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *AssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider.Client, got: %T. Please report this issue.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *AssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ModifyPlan marks the attributes the server recomputes on an in-place
// update as Unknown when the change that would affect them is actually
// present in the plan. Without this, the per-attribute UseStateForUnknown
// modifiers (needed so a no-op plan doesn't show these as "known after
// apply" on every run) would also carry the *old* value forward into an
// Update plan, which then mismatches whatever Update() actually returns
// from the server ("Provider produced inconsistent result after apply").
func (r *AssetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Create (no prior state) and Delete (no planned state) both leave
	// these fields fully unknown/absent already -- nothing to adjust.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan, config assetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameChanged := plan.Name.ValueString() != state.Name.ValueString()
	fileChanged := plan.Source.ValueString() != state.Source.ValueString() ||
		plan.SourceHash.ValueString() != state.SourceHash.ValueString()

	if nameChanged {
		plan.URL = types.StringUnknown()
	}
	if fileChanged {
		plan.SHA256 = types.StringUnknown()
		plan.Integrity = types.StringUnknown()
		plan.SizeBytes = types.Int64Unknown()
		// Only when content_type isn't explicitly pinned in config --
		// otherwise the configured value is already what will be sent.
		if config.ContentType.IsNull() {
			plan.ContentType = types.StringUnknown()
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func forceDownloadField(v types.Bool) *bool {
	if v.IsUnknown() || v.IsNull() {
		return nil
	}
	value := v.ValueBool()
	return &value
}

func applyAssetToModel(asset *Asset, model *assetResourceModel) {
	model.ID = types.StringValue(asset.ID)
	model.Name = types.StringValue(asset.Name)
	model.ContentType = types.StringValue(asset.ContentType)
	model.ForceDownload = types.BoolValue(asset.ForceDownload)
	model.URL = types.StringValue(asset.URL)
	model.SHA256 = types.StringValue(asset.SHA256)
	model.Integrity = types.StringValue(asset.Integrity)
	model.SizeBytes = types.Int64Value(asset.SizeBytes)
	model.CreatedAt = types.StringValue(asset.CreatedAt)
}

func (r *AssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan assetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourcePath := plan.Source.ValueString()
	fileBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read source file", err.Error())
		return
	}

	asset, err := r.client.CreateAsset(filepath.Base(sourcePath), fileBytes, assetFields{
		Name:          plan.Name.ValueString(),
		ContentType:   plan.ContentType.ValueString(),
		ForceDownload: forceDownloadField(plan.ForceDownload),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create ContentFlow asset", err.Error())
		return
	}

	applyAssetToModel(asset, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state assetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	asset, err := r.client.GetAsset(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read ContentFlow asset", err.Error())
		return
	}
	if asset == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	applyAssetToModel(asset, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state assetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only re-upload the file when the caller signaled a content change
	// (source_hash) or pointed `source` at a different path -- otherwise
	// this is a metadata-only update (rename / content-type / force_download).
	fileChanged := plan.Source.ValueString() != state.Source.ValueString() ||
		plan.SourceHash.ValueString() != state.SourceHash.ValueString()

	var fileBytes []byte
	fileName := ""
	if fileChanged {
		sourcePath := plan.Source.ValueString()
		var err error
		fileBytes, err = os.ReadFile(sourcePath)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read source file", err.Error())
			return
		}
		fileName = filepath.Base(sourcePath)
	}

	asset, err := r.client.UpdateAsset(state.ID.ValueString(), fileName, fileBytes, assetFields{
		Name:          plan.Name.ValueString(),
		ContentType:   plan.ContentType.ValueString(),
		ForceDownload: forceDownloadField(plan.ForceDownload),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update ContentFlow asset", err.Error())
		return
	}

	applyAssetToModel(asset, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AssetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state assetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAsset(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete ContentFlow asset", err.Error())
	}
}
