package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
)

const internalErrorClarification = "An unexpected error was encountered while validating. This is always an error in the provider. Please report the following to the provider developer:\n\n"

type oneofValidator struct {
	names   []string
	nameMap map[string]map[string]string
}

var _ validator.String = (*oneofValidator)(nil)
var _ validator.Bool = (*oneofValidator)(nil)
var _ validator.Dynamic = (*oneofValidator)(nil)
var _ validator.Float64 = (*oneofValidator)(nil)
var _ validator.Int64 = (*oneofValidator)(nil)
var _ validator.Number = (*oneofValidator)(nil)
var _ validator.Object = (*oneofValidator)(nil)

func OneofValidator(
	names []string,
	nameMap map[string]map[string]string,
) *oneofValidator {
	return &oneofValidator{
		names:   names,
		nameMap: nameMap,
	}
}

func (v *oneofValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *oneofValidator) MarkdownDescription(_ context.Context) string {
	return "Only one of the following fields can be set:\n\n* " +
		strings.Join(v.names, "\n* ")
}

func (v *oneofValidator) validate(
	ctx context.Context, currentPath path.Path, config tfsdk.Config,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	currentAttrStep, _ := currentPath.Steps().LastStep()
	currentAttrName, ok := currentAttrStep.(path.PathStepAttributeName)
	if !ok {
		diags.AddAttributeError(
			currentPath,
			"path is not attribute",
			fmt.Sprintf(
				internalErrorClarification+"Path %s is not a path to an "+
					"attribute, oneof validator expects attributes as targets.",
				currentPath,
			),
		)
		return diags
	}

	obj := types.Object{}
	innerDiag := config.GetAttribute(ctx, currentPath.ParentPath(), &obj)
	diags.Append(innerDiag...)
	if innerDiag.HasError() {
		return diags
	}
	parentPathStr := currentPath.ParentPath().String()
	attrs := obj.Attributes()
	currentAttr, ok := attrs[string(currentAttrName)]
	if !ok {
		if parentPathStr == "" {
			parentPathStr = "current resource"
		}
		diags.AddAttributeError(
			currentPath,
			"couldn't find attribute in parent object",
			fmt.Sprintf(
				internalErrorClarification+
					"Couldn't find %s in attributes of %s extracted as "+
					"types.Object",
				currentAttrName, parentPathStr,
			),
		)
		return diags
	}
	currentAttr, _, innerDiags := ctypes.UnwrapDynamic(ctx, currentAttr)
	diags.Append(innerDiags...)
	// skip unset values
	if currentAttr.IsNull() || currentAttr.IsUnknown() {
		return diags
	}
	if parentPathStr != "" {
		parentPathStr = parentPathStr + "."
	}
	for _, otherName := range v.names {
		if otherName == string(currentAttrName) {
			continue
		}
		otherAttr, ok := attrs[otherName]
		if !ok {
			continue
		}
		otherAttr, _, innerDiags := ctypes.UnwrapDynamic(ctx, otherAttr)
		diags.Append(innerDiags...)
		if otherAttr.IsNull() || otherAttr.IsUnknown() {
			continue
		}
		diags.AddAttributeError(
			currentPath,
			"attribute collision",
			fmt.Sprintf(
				"Attribute %s must not be set together with %s%s, as they "+
					"are parts of one one-of",
				currentPath, parentPathStr, otherName,
			),
		)
	}

	return diags
}

func (v *oneofValidator) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *oneofValidator) ValidateBool(
	ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *oneofValidator) ValidateDynamic(
	ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *oneofValidator) ValidateFloat64(
	ctx context.Context, req validator.Float64Request, resp *validator.Float64Response,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
func (v *oneofValidator) ValidateInt64(
	ctx context.Context, req validator.Int64Request, resp *validator.Int64Response,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
func (v *oneofValidator) ValidateNumber(
	ctx context.Context, req validator.NumberRequest, resp *validator.NumberResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
func (v *oneofValidator) ValidateObject(
	ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
