package main

import (
	"github.com/abergmeier/terraform-provider-exasol/internal/resourceprovider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: resourceprovider.Provider,
	})
}
