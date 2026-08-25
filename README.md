# terraform-provider-contentflow

A Terraform provider for managing assets on a self-hosted
[ContentFlow](https://github.com/xXxNIKIxXx/ContentFlow) dashboard.
`terraform apply` uploads a file and returns the URL it is served at;
`terraform destroy` removes it.

The provider communicates with the dashboard's token-authenticated
`/api/v1` JSON API.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- A ContentFlow dashboard with `DASHBOARD_API_TOKEN` configured

## Installation

```hcl
terraform {
  required_providers {
    contentflow = {
      source  = "DEV-F0x/contentflow"
      version = "~> 1.0"
    }
  }
}
```

## Usage

```hcl
provider "contentflow" {
  endpoint  = "https://dashboard.example.com"
  api_token = var.contentflow_api_token # or set CONTENTFLOW_API_TOKEN
}

resource "contentflow_asset" "logo" {
  name        = "logo.svg"
  source      = "${path.module}/logo.svg"
  source_hash = filesha256("${path.module}/logo.svg")
}

output "logo_url" {
  value = contentflow_asset.logo.url
}
```

A runnable copy of this configuration is provided in [`examples/`](examples/).

Full provider and resource documentation, including all arguments and
attributes, is published on the
[Terraform Registry](https://registry.terraform.io/providers/DEV-F0x/contentflow/latest/docs).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for building from source, running
against a local ContentFlow instance, and the release process.

## License

[MIT](LICENSE)
