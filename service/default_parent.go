package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nebius/gosdk/nid"
	"github.com/nebius/terraform-provider-nebius/provider"
)

type defaultParent struct {
	provider            provider.Provider
	parentResourceTypes []string
}

var _ defaults.String = (*defaultParent)(nil)

func NewDefaultParent(
	p provider.Provider,
	parentResourceTypes []string,
) defaults.String {
	return &defaultParent{
		provider:            p,
		parentResourceTypes: slices.Clone(parentResourceTypes),
	}
}

func (d *defaultParent) Description(_ context.Context) string {
	return "Uses the provider parent ID when its resource type is allowed for this resource."
}

func (d *defaultParent) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (d *defaultParent) DefaultString(
	_ context.Context,
	req defaults.StringRequest,
	resp *defaults.StringResponse,
) {
	resp.PlanValue, resp.Diagnostics = compatibleDefaultParent(
		d.provider.DefaultParentID(),
		d.parentResourceTypes,
		req.Path,
	)
}

func compatibleDefaultParent(
	parentID types.String,
	parentResourceTypes []string,
	parentPath path.Path,
) (types.String, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if len(parentResourceTypes) == 0 {
		diags.AddAttributeError(
			parentPath,
			"parent not set",
			"Set parent_id explicitly in this configuration.",
		)
		return types.StringNull(), diags
	}
	if parentID.IsUnknown() {
		return types.StringUnknown(), diags
	}
	if parentID.IsNull() {
		diags.AddAttributeError(
			parentPath,
			"parent_id is not configured",
			"Set parent_id in this configuration or configure parent_id in the provider.",
		)
		return types.StringNull(), diags
	}

	parsed, err := nid.Parse(parentID.ValueString())
	if err != nil {
		diags.AddAttributeError(
			parentPath,
			"provider parent_id is invalid",
			fmt.Sprintf(
				"Provider parent_id %q is not a valid Nebius ID: %s. Set parent_id explicitly in this configuration.",
				parentID.ValueString(),
				err,
			),
		)
		return types.StringNull(), diags
	}

	parentType := string(parsed.Type)
	if slices.Contains(parentResourceTypes, "*") ||
		slices.Contains(parentResourceTypes, parentType) {
		return parentID, diags
	}

	detail := fmt.Sprintf(
		"Provider parent_id has type %q, but this object accepts parent types: %s. Set parent_id explicitly in this configuration.",
		parentType,
		strings.Join(parentResourceTypes, ", "),
	)
	diags.AddAttributeError(
		parentPath,
		"provider parent_id has an incompatible type",
		detail,
	)
	return types.StringNull(), diags
}
