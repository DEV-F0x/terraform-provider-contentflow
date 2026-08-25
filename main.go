package main

// Deliberately NOT a tracked `go tool` dependency (`go get -tool`): that
// pulls tfplugindocs's own dependency tree (go-git, various crypto libs,
// ...) straight into this module's go.sum, where Dependabot flags it
// alongside the provider's actual runtime dependencies even though none of
// it ships in the built provider binary. `go run <pkg>@<version>` runs it
// from its own ephemeral module instead, touching nothing here.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/DEV-F0x/terraform-provider-contentflow/internal/provider"
)

// version is set at build time via:
//
//	go build -ldflags "-X main.version=1.0.0"
//
// Left as "dev" for local builds used with Terraform's dev_overrides.
var version string = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Only meaningful if this is ever published to a registry; local
		// use goes through Terraform's dev_overrides instead, which
		// bypasses registry resolution entirely (see the README).
		Address: "registry.terraform.io/DEV-F0x/contentflow",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
