package main

//go:generate go tool tfplugindocs generate

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
