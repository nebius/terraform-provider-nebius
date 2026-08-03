package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ValidateLabels(
	labels types.Map,
	labelsPath path.Path,
	requireKnown bool,
) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if labels.IsNull() {
		return diags
	}
	if labels.IsUnknown() {
		if requireKnown {
			diags.AddAttributeError(
				labelsPath,
				"labels are unknown",
				"Labels must be known before they can be sent to the API.",
			)
		}
		return diags
	}

	for key, value := range labels.Elements() {
		valuePath := labelsPath.AtMapKey(key)
		if value.IsNull() {
			diags.AddAttributeError(
				valuePath,
				"label value is null",
				fmt.Sprintf("Label %q must have a non-null string value.", key),
			)
		}
		if requireKnown && value.IsUnknown() {
			diags.AddAttributeError(
				valuePath,
				"label value is unknown",
				fmt.Sprintf("Label %q must be known before it can be sent to the API.", key),
			)
		}
	}
	return diags
}
