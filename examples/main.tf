terraform {
  required_providers {
    contentflow = {
      source = "DEV-F0x/contentflow"
    }
  }
}

provider "contentflow" {
  # Or set CONTENTFLOW_ENDPOINT / CONTENTFLOW_API_TOKEN instead.
  endpoint  = "http://localhost:5001"
  api_token = var.contentflow_api_token
}

variable "contentflow_api_token" {
  type      = string
  sensitive = true
}

resource "contentflow_asset" "hello" {
  name        = "hello.txt"
  source      = "${path.module}/hello.txt"
  source_hash = filesha256("${path.module}/hello.txt")
}

output "hello_url" {
  value = contentflow_asset.hello.url
}
