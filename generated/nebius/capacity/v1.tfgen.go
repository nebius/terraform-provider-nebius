package capacity

import (
	child "github.com/nebius/terraform-provider-nebius/generated/nebius/capacity/v1"
)

func init() { //nolint: gochecknoinits // registry dynamic registration
	ResourceFactories = append(ResourceFactories, child.ResourceFactories...)
	DatasourceFactories = append(DatasourceFactories, child.DatasourceFactories...)
	EphemeralFactories = append(EphemeralFactories, child.EphemeralFactories...)
}
