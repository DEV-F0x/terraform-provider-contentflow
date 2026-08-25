# terraform-provider-contentflow

A Terraform provider for managing assets on a self-hosted
[ContentFlow](https://github.com/xXxNIKIxXx/ContentFlow) dashboard.
`terraform apply` uploads a file and returns the real, working URL it's
served at; `terraform destroy` removes it.

It talks to the dashboard's token-authenticated `/api/v1` JSON API (see
[`services/dashboard/app/api_routes.py`](https://github.com/xXxNIKIxXx/ContentFlow/blob/main/services/dashboard/app/api_routes.py)
and the ["JSON API" section](https://github.com/xXxNIKIxXx/ContentFlow#json-api-for-terraform--scripts)
of ContentFlow's README) rather than the dashboard's cookie-session login.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- Go >= 1.25, only if building from source
- A ContentFlow dashboard with `DASHBOARD_API_TOKEN` set

## Installing

Once published to the Terraform Registry:

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

Until then (or to run a local build), see
["Using a local build"](#using-a-local-build) below.

## Example usage

```hcl
provider "contentflow" {
  endpoint  = "https://dashboard.example.com"
  api_token = var.contentflow_api_token # or set CONTENTFLOW_API_TOKEN instead
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

A complete, runnable copy of this lives in [`examples/`](examples/).

### Provider configuration

| Attribute | Env var fallback | Required |
|---|---|---|
| `endpoint` | `CONTENTFLOW_ENDPOINT` | yes |
| `api_token` | `CONTENTFLOW_API_TOKEN` | yes |

### `contentflow_asset` resource

| Attribute | Type | Description |
|---|---|---|
| `name` | Required, string | Served name / storage key -- the asset ends up at `<url>/files/<name>`. |
| `source` | Required, string | Path to the local file to upload. |
| `source_hash` | Optional, string | Set to `filesha256(path)` (or similar) so Terraform can detect content changes -- it can't tell from `source` alone, since that's just a path string. |
| `content_type` | Optional+Computed, string | MIME type override. Auto-detected from the file extension when omitted. |
| `force_download` | Optional, bool (default `false`) | When true, served with `Content-Disposition: attachment`. |
| `id` | Computed | The asset's database id. |
| `url` | Computed | The real, working URL the asset is served at. |
| `sha256` / `integrity` | Computed | Hash / Subresource Integrity string of the uploaded content. |
| `size_bytes` | Computed | |
| `created_at` | Computed | |

Renaming (`name`) or replacing the file (`source` + `source_hash`) updates
the asset in place -- same `id`, no destroy/recreate.
`terraform import contentflow_asset.<name> <asset-id>` is supported (the
asset id, as shown in the dashboard or `GET /api/v1/assets`).

## Using a local build

```bash
go build -o terraform-provider-contentflow .
```

Point Terraform's CLI config at the binary instead of the registry, via
`~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "DEV-F0x/contentflow" = "/absolute/path/to/terraform-provider-contentflow"
  }
  direct {}
}
```

(The path is to the *directory* containing the binary, not the binary
itself.) With this in place, `terraform init` skips provider installation
entirely -- it prints a warning that dev overrides are active, which is
expected -- and every `terraform` command uses your local build directly.
Rebuild and rerun; no `terraform init` needed between changes.

## Publishing a release

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which uses
[GoReleaser](https://goreleaser.com) (`.goreleaser.yml`) to build binaries
for every supported OS/arch, checksum and GPG-sign them, and attach
everything -- plus `terraform-registry-manifest.json` -- to a GitHub
Release. Two repository secrets are required first:

| Secret | Value |
|---|---|
| `GPG_PRIVATE_KEY` | `gpg --armor --export-secret-keys <key-id>` |
| `PASSPHRASE` | the key's passphrase |

To connect a release to the Terraform Registry: sign in at
[registry.terraform.io](https://registry.terraform.io) with GitHub,
**Publish -> Provider**, select this repository, and add the matching GPG
**public** key under Settings -> Signing Keys if it isn't there already.
The registry then tracks every `vX.Y.Z` tag automatically.

## Documentation

The registry renders its docs pages straight from [`docs/`](docs/) in the
repository -- if that directory is missing or out of date, the registry
shows "Documentation Unavailable" for that version instead of erroring, so
it's easy to miss. Regenerate it from the provider schema + the example
snippets in `examples/provider/` and `examples/resources/<name>/` with:

```bash
go generate ./...
```

(via [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs),
invoked by pinned version straight from `main.go`'s `go:generate` directive
-- deliberately *not* a tracked `go tool` dependency in `go.mod`, since that
would pull its own dependency tree, e.g. go-git, into this module's
`go.sum`, where Dependabot flags it alongside the provider's real runtime
dependencies even though none of it ships in the built binary.) Commit the
result and include it in the next `vX.Y.Z` tag; docs are tied to the release
they came from, so pushing to `main` alone doesn't update what an
already-published version shows.

## Project layout

```
main.go                              providerserver.Serve entry point; go:generate directive for docs
internal/provider/
  provider.go                        provider schema + configuration
  client.go                          REST client for /api/v1/assets
  asset_resource.go                  contentflow_asset: CRUD + plan logic
examples/                            runnable example configuration
examples/provider/                   snippet embedded in docs/index.md
examples/resources/contentflow_asset/  snippets embedded in docs/resources/asset.md
docs/                                generated registry documentation (see above)
.goreleaser.yml                      build/sign/release config
terraform-registry-manifest.json     protocol version for the registry
.github/workflows/release.yml        runs GoReleaser on a vX.Y.Z tag push
```

## License

[MIT](LICENSE)
