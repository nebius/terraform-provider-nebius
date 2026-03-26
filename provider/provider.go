package provider

import "github.com/nebius/gosdk"

type Provider interface {
	SDK() *gosdk.SDK
	WriteOnlyFieldsSupported() bool
}
