package main

import (
	"context"
	"flag"
	"log"

	"github.com/hadamrd/terraform-provider-bbox/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is filled by ldflags at release time.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run in debug mode for use with a debugger like Delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/hadamrd/bbox",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
