terraform {
  required_providers {
    contentflow = {
      source = "DEV-F0x/contentflow"
    }
  }
}

provider "contentflow" {
  # Base URL of your ContentFlow dashboard. Or set CONTENTFLOW_ENDPOINT.
  endpoint = "https://dashboard.example.com"

  # Bearer token -- must match DASHBOARD_API_TOKEN on the server.
  # Or set CONTENTFLOW_API_TOKEN instead of hardcoding it here.
  api_token = var.contentflow_api_token
}

variable "contentflow_api_token" {
  type      = string
  sensitive = true
}
