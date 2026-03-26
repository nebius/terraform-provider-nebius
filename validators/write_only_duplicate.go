package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type writeOnlyDuplicateValidator struct{}

var _ validator.Bool = (*writeOnlyDuplicateValidator)(nil)
var _ validator.Dynamic = (*writeOnlyDuplicateValidator)(nil)
var _ validator.Float64 = (*writeOnlyDuplicateValidator)(nil)
var _ validator.Int64 = (*writeOnlyDuplicateValidator)(nil)
var _ validator.List = (*writeOnlyDuplicateValidator)(nil)
var _ validator.Map = (*writeOnlyDuplicateValidator)(nil)
var _ validator.Number = (*writeOnlyDuplicateValidator)(nil)
var _ validator.Object = (*writeOnlyDuplicateValidator)(nil)
var _ validator.String = (*writeOnlyDuplicateValidator)(nil)

// WriteOnlyDuplicateValidator warns when the state-saved duplicate
// of a write-only field is used.
func WriteOnlyDuplicateValidator() *writeOnlyDuplicateValidator {
	return &writeOnlyDuplicateValidator{}
}

func (v *writeOnlyDuplicateValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *writeOnlyDuplicateValidator) MarkdownDescription(_ context.Context) string {
	return "Warn when the state-saved duplicate of a write-only field is used."
}

func (v *writeOnlyDuplicateValidator) ValidateBool(
	ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateDynamic(
	ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateFloat64(
	ctx context.Context, req validator.Float64Request, resp *validator.Float64Response,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateInt64(
	ctx context.Context, req validator.Int64Request, resp *validator.Int64Response,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateList(
	ctx context.Context, req validator.ListRequest, resp *validator.ListResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateMap(
	ctx context.Context, req validator.MapRequest, resp *validator.MapResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateNumber(
	ctx context.Context, req validator.NumberRequest, resp *validator.NumberResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateObject(
	ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *writeOnlyDuplicateValidator) validate(
	ctx context.Context,
	value attr.Value,
	currentPath path.Path,
) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if value == nil {
		return diags
	}

	switch val := value.(type) {
	case basetypes.BoolValuable:
		boolVal, innerDiags := val.ToBoolValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || boolVal.IsNull() || boolVal.IsUnknown() {
			return diags
		}
	case basetypes.DynamicValuable:
		dynVal, innerDiags := val.ToDynamicValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || dynVal.IsNull() || dynVal.IsUnknown() {
			return diags
		}
	case basetypes.Float64Valuable:
		floatVal, innerDiags := val.ToFloat64Value(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || floatVal.IsNull() || floatVal.IsUnknown() {
			return diags
		}
	case basetypes.Int64Valuable:
		intVal, innerDiags := val.ToInt64Value(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || intVal.IsNull() || intVal.IsUnknown() {
			return diags
		}
	case basetypes.ListValuable:
		listVal, innerDiags := val.ToListValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || listVal.IsNull() || listVal.IsUnknown() {
			return diags
		}
	case basetypes.MapValuable:
		mapVal, innerDiags := val.ToMapValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || mapVal.IsNull() || mapVal.IsUnknown() {
			return diags
		}
	case basetypes.NumberValuable:
		numberVal, innerDiags := val.ToNumberValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || numberVal.IsNull() || numberVal.IsUnknown() {
			return diags
		}
	case basetypes.ObjectValuable:
		objVal, innerDiags := val.ToObjectValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || objVal.IsNull() || objVal.IsUnknown() {
			return diags
		}
	case basetypes.StringValuable:
		strVal, innerDiags := val.ToStringValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || strVal.IsNull() || strVal.IsUnknown() {
			return diags
		}
	default:
		if value.IsNull() || value.IsUnknown() {
			return diags
		}
	}

	diags.AddAttributeWarning(
		currentPath,
		"use write-only field",
		fmt.Sprintf(
			"Insecure state-saved field %s is used. Use more secure "+
				"write-only field %s instead. These fields are more secure "+
				" starting from Terraform 1.11.0, in previous versions they "+
				" will be as insecure as the state-saved fields without "+
				" any warnings.",
			currentPath,
			pathToWOPath(currentPath),
		),
	)

	return diags
}

func pathToWOPath(p path.Path) path.Path {
	ret := path.Root("sensitive")
	for _, s := range p.Steps() {
		switch typed := s.(type) {
		case path.PathStepAttributeName:
			ret = ret.AtName(string(typed))
		case path.PathStepElementKeyInt:
			ret = ret.AtListIndex(int(typed))
		case path.PathStepElementKeyString:
			ret = ret.AtMapKey(string(typed))
		}
	}
	return ret
}
