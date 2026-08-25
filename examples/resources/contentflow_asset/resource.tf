resource "contentflow_asset" "logo" {
  name   = "logo.png"
  source = "${path.module}/logo.png"

  # Required to detect when the file's *contents* change -- Terraform can't
  # tell from `source` alone, since that's just a path string.
  source_hash = filesha256("${path.module}/logo.png")
}

output "logo_url" {
  value = contentflow_asset.logo.url
}
