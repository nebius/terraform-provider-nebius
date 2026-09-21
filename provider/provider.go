package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nebius/gosdk"
)

type Provider interface {
	SDK() *gosdk.SDK
	DefaultLabels() types.Map
	DefaultParentID() types.String
	WriteOnlyFieldsSupported() bool
	PreflightChecksDisabled() bool
}
