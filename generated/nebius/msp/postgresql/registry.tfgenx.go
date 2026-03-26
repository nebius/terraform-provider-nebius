package postgresql

import (
	datasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	resource "github.com/hashicorp/terraform-plugin-framework/resource"
	ephemeral "github.com/hashicorp/terraform-plugin-framework/ephemeral"
	provider "github.com/nebius/terraform-provider-nebius/provider"
)

var ResourceFactories = []func(provider.Provider) resource.Resource{}
var DatasourceFactories = []func(provider.Provider) datasource.DataSource{}
var EphemeralFactories = []func(provider.Provider) ephemeral.EphemeralResource{}
