package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// main is the entrypoint for the Terraform provider plugin.  It
// delegates to the plugin framework's providerserver to serve the
// provider over RPC.  The debug flag enables support for
// debugging tools such as delve when set.
func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	// Serve the provider.  The address identifies the provider in
	// Terraform configurations.  When publishing to the Terraform
	// Registry this should follow the registry namespace pattern
	// (e.g. registry.terraform.io/tenable/tenablevm).  For local
	// development any address may be used as long as it matches the
	// CLI configuration.
	err := providerserver.Serve(
		context.Background(),
		func() provider.Provider { return NewProvider("dev") },
		providerserver.ServeOpts{
			Address: "registry.terraform.io/tenable/tenablevm",
			Debug:   debug,
		},
	)
	if err != nil {
		log.Fatal(err.Error())
	}
}
