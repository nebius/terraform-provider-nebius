package service

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/nebius/gosdk/constants"
	"github.com/nebius/terraform-provider-nebius/provider"
)

func mergeDefaultLabels(
	defaultLabels types.Map,
	resourceLabels types.Map,
) (types.Map, diag.Diagnostics) {
	if defaultLabels.IsUnknown() || resourceLabels.IsUnknown() {
		return types.MapUnknown(types.StringType), nil
	}

	elements := map[string]attr.Value{}
	if !defaultLabels.IsNull() {
		maps.Copy(elements, defaultLabels.Elements())
	}
	if !resourceLabels.IsNull() {
		maps.Copy(elements, resourceLabels.Elements())
	}
	return types.MapValue(types.StringType, elements)
}

// Default labels can change after the framework marks computed fields as unknown.
// Invalidate mutable outputs when the effective labels change during an update.
func (r *commonResource) invalidateLabelUpdateOutputs(
	ctx context.Context,
	state tfsdk.State,
	plan *tfsdk.Plan,
	labelsAll types.Map,
) diag.Diagnostics {
	if state.Raw.IsNull() {
		return nil
	}
	var previousLabels types.Map
	diags := state.GetAttribute(ctx, path.Root(constants.FieldLabelsAll), &previousLabels)
	if diags.HasError() || labelsAll.Equal(previousLabels) {
		return diags
	}
	for _, name := range []string{
		constants.FieldResourceVersion,
		constants.FieldUpdatedAt,
		constants.FieldStatus,
	} {
		attribute, ok := r.resourceSchema.Attributes[name]
		if !ok || !attribute.IsComputed() || attribute.IsOptional() {
			continue
		}
		attributeType := attribute.GetType()
		unknown, err := attributeType.ValueFromTerraform(
			ctx, tftypes.NewValue(attributeType.TerraformType(ctx), tftypes.UnknownValue),
		)
		if err != nil {
			diags.AddAttributeError(
				path.Root(name),
				"Cannot plan computed attribute",
				fmt.Sprintf("Cannot create an unknown value for this computed attribute: %s.", err),
			)
			return diags
		}
		diags.Append(plan.SetAttribute(ctx, path.Root(name), unknown)...)
		if diags.HasError() {
			return diags
		}
	}
	return diags
}

func inferImportedResourceLabels(
	defaultLabels types.Map,
	remoteLabels map[string]string,
) (types.Map, diag.Diagnostics) {
	diags := provider.ValidateLabels(
		defaultLabels,
		path.Root("default_labels"),
		true,
	)
	if diags.HasError() {
		return types.MapNull(types.StringType), diags
	}

	defaultElements := map[string]attr.Value{}
	if !defaultLabels.IsNull() {
		defaultElements = defaultLabels.Elements()
	}
	inferred := make(map[string]attr.Value, len(remoteLabels))
	for key, remoteValue := range remoteLabels {
		defaultValue, ok := defaultElements[key].(types.String)
		if !ok || defaultValue.ValueString() != remoteValue {
			inferred[key] = types.StringValue(remoteValue)
		}
	}
	if len(inferred) == 0 {
		return types.MapNull(types.StringType), diags
	}
	ret, innerDiags := types.MapValue(types.StringType, inferred)
	diags.Append(innerDiags...)
	return ret, diags
}

func objectWithResourceLabels(
	ctx context.Context,
	data types.Object,
	labels types.Map,
) (types.Object, diag.Diagnostics) {
	attrs := data.Attributes()
	if _, ok := attrs[constants.FieldLabels]; !ok {
		diags := diag.Diagnostics{}
		diags.AddError(
			"labels not found",
			"Cannot set inferred imported labels because labels are missing from state.",
		)
		return data, diags
	}
	attrs[constants.FieldLabels] = labels
	return basetypes.NewObjectValue(data.AttributeTypes(ctx), attrs)
}
