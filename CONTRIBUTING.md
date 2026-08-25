# Contributing

## Building from source

Requires Go >= 1.25.

```bash
go build -o terraform-provider-contentflow .
```

## Running against a local build

Point Terraform's CLI configuration at the built binary instead of the
registry. Add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "DEV-F0x/contentflow" = "/absolute/path/to/terraform-provider-contentflow"
  }
  direct {}
}
```

The path must be the directory containing the binary, not the binary
itself. With this configured, `terraform init` skips provider installation
and every `terraform` command uses the local build directly; rebuild and
rerun without needing to reinitialize.

## Project layout

```
main.go                                providerserver.Serve entry point
internal/provider/
  provider.go                          provider schema and configuration
  client.go                            REST client for /api/v1/assets
  asset_resource.go                    contentflow_asset: CRUD and plan logic
examples/                              example configurations, also embedded in docs/
docs/                                  generated Terraform Registry documentation
.goreleaser.yml                        release build configuration
terraform-registry-manifest.json       supported protocol version
.github/workflows/release.yml          release workflow
```

## Documentation

Registry documentation is generated from the provider schema and the
example snippets under `examples/provider/` and
`examples/resources/<name>/`:

```bash
go generate ./...
```

This runs [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs)
by pinned version rather than as a tracked module dependency, so its
toolchain does not appear in this project's own dependency graph.

Commit the regenerated `docs/` and include it in the next release tag.
Documentation is tied to the release it was published with, so changes to
`main` alone do not update an already-published version.

## Releasing

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which uses
[GoReleaser](https://goreleaser.com) to build binaries for every supported
platform, checksum and sign them, and publish a GitHub Release.

### Required repository secrets

| Secret | Value |
|---|---|
| `GPG_PRIVATE_KEY` | `gpg --armor --export-secret-keys <key-id>` |
| `PASSPHRASE` | the signing key's passphrase |

### Publishing to the Terraform Registry

1. Sign in at [registry.terraform.io](https://registry.terraform.io) with
   GitHub.
2. Select **Publish -> Provider** and choose this repository.
3. Add the corresponding GPG public key under **Settings -> Signing Keys**
   if it is not already present.

Once published, the registry tracks new `vX.Y.Z` tags automatically.

## Verifying changes

```bash
go build -o terraform-provider-contentflow .
go vet ./...
gofmt -l .
govulncheck ./...
```
