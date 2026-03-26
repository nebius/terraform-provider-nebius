package validators

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"buf.build/go/protovalidate"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	maskprotovalidate "github.com/nebius/gosdk/proto/fieldmask/mask/protovalidate"
	"github.com/nebius/terraform-provider-nebius/conversion"
)

var validatorEngine protovalidate.Validator
var engineMutex = sync.Mutex{}

type protoFieldValidator struct {
	parent  proto.Message
	name    protoreflect.Name
	tfName  string
	nameMap map[string]map[string]string
}

var _ validator.String = (*protoFieldValidator)(nil)
var _ validator.Bool = (*protoFieldValidator)(nil)
var _ validator.Dynamic = (*protoFieldValidator)(nil)
var _ validator.Float64 = (*protoFieldValidator)(nil)
var _ validator.Int64 = (*protoFieldValidator)(nil)
var _ validator.Number = (*protoFieldValidator)(nil)
var _ validator.Object = (*protoFieldValidator)(nil)
var _ validator.List = (*protoFieldValidator)(nil)
var _ validator.Map = (*protoFieldValidator)(nil)

func ProtoFieldValidator(
	parent proto.Message,
	name protoreflect.Name,
	tfName string,
	nameMap map[string]map[string]string,
) *protoFieldValidator {
	return &protoFieldValidator{
		parent:  parent,
		name:    name,
		tfName:  tfName,
		nameMap: nameMap,
	}
}

func (v *protoFieldValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *protoFieldValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf(
		"Validate the field %q inside message %q using protovalidate."+
			" For message fields, filter the unknowns.",
		v.name,
		v.parent.ProtoReflect().Descriptor().FullName(),
	)
}

func (v *protoFieldValidator) validate(
	ctx context.Context, currentPath path.Path, config tfsdk.Config,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	currentAttrStep, _ := currentPath.Steps().LastStep()
	currentAttrName, ok := currentAttrStep.(path.PathStepAttributeName)
	if !ok {
		diags.AddAttributeError(
			currentPath,
			"path is not attribute",
			fmt.Sprintf(internalErrorClarification+
				"Path %s is not a path to an attribute, proto "+
				"validator expects attributes as targets", currentPath),
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
			fmt.Sprintf(internalErrorClarification+
				"Couldn't find %s in attributes of %s extracted as types.Object",
				currentAttrName, parentPathStr,
			),
		)
		return diags
	}
	// skip unset values
	if currentAttr.IsNull() || currentAttr.IsUnknown() {
		return diags
	}

	converterObj, innerDiag := types.ObjectValue(
		map[string]attr.Type{
			v.tfName: currentAttr.Type(ctx),
		},
		map[string]attr.Value{
			v.tfName: currentAttr,
		},
	)
	diags.Append(innerDiag...)
	if innerDiag.HasError() {
		return diags
	}
	target := v.parent.ProtoReflect().New().Interface()
	unk, innerDiag := conversion.MessageFromTF(
		ctx, converterObj, target, v.nameMap,
	)
	diags.Append(innerDiag...)
	if innerDiag.HasError() {
		return diags
	}
	err := v.validateMessage(target)
	if err != nil {
		if parentPathStr != "" {
			parentPathStr = parentPathStr + "."
		}
		var valError *protovalidate.ValidationError
		if errors.As(err, &valError) {
			violations := valError.ToProto()

		violationChecker:
			for _, violation := range violations.GetViolations() {
				field := violation.GetField()
				if field == nil || len(field.GetElements()) == 0 ||
					field.GetElements()[0].GetFieldName() != string(v.name) {
					continue
				}
				violationPath, convErr := maskprotovalidate.FieldPathFromProto(field)
				if convErr != nil {
					diags.AddAttributeError(
						currentPath,
						"invalid violation field path",
						fmt.Sprintf(
							internalErrorClarification+
								"Failed to convert protovalidate field path to fieldmask path: %s",
							convErr,
						),
					)
					continue
				}

				if violationPath != nil && unk != nil &&
					(unk.IsEmpty() || violationPath.MatchesResetMask(unk)) {
					tflog.Info(ctx,
						"Validation error for unknown value",
						map[string]any{
							"unknown_path":         violationPath.String(),
							"violation_constraint": violation.GetRuleId(),
							"violation_message":    violation.GetMessage(),
							"violation_path":       violation.GetField().String(),
							"attribute_path":       currentPath.String(),
						},
					)
					continue violationChecker
				}
				violationPathString := violation.GetField().String()
				if violationPath != nil {
					violationPathString = violationPath.String()
				}
				diags.AddAttributeError(
					currentPath,
					"attribute validation error",
					fmt.Sprintf(
						"Attribute %s%s constraint %s not met: %s",
						parentPathStr,
						violationPathString,
						violation.GetRuleId(),
						violation.GetMessage(),
					),
				)
			}
		} else {
			diags.AddAttributeError(
				currentPath,
				"protovalidate error",
				fmt.Sprintf(
					internalErrorClarification+
						"While validating %q field, protovalidate raised the "+
						"following error: %s",
					currentPath,
					err,
				),
			)
		}
	}

	return diags
}

// Not using once.Do, for the sake of error recovery
func getEngine() (protovalidate.Validator, error) {
	engineMutex.Lock()
	defer engineMutex.Unlock()

	if validatorEngine == nil {
		ve, err := protovalidate.New()
		if err != nil {
			return nil, err
		}
		validatorEngine = ve
	}
	return validatorEngine, nil
}

func (v *protoFieldValidator) validateMessage(msg proto.Message) error {
	eng, err := getEngine()
	if err != nil {
		return fmt.Errorf("creating validator engine: %w", err)
	}
	err = eng.Validate(msg)
	if err != nil {
		return fmt.Errorf("validation: %w", err)
	}
	return nil
}

func (v *protoFieldValidator) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *protoFieldValidator) ValidateBool(
	ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *protoFieldValidator) ValidateDynamic(
	ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *protoFieldValidator) ValidateFloat64(
	ctx context.Context, req validator.Float64Request, resp *validator.Float64Response,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
func (v *protoFieldValidator) ValidateInt64(
	ctx context.Context, req validator.Int64Request, resp *validator.Int64Response,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
func (v *protoFieldValidator) ValidateNumber(
	ctx context.Context, req validator.NumberRequest, resp *validator.NumberResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
func (v *protoFieldValidator) ValidateObject(
	ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *protoFieldValidator) ValidateList(
	ctx context.Context, req validator.ListRequest, resp *validator.ListResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}

func (v *protoFieldValidator) ValidateMap(
	ctx context.Context, req validator.MapRequest, resp *validator.MapResponse,
) {
	diags := v.validate(ctx, req.Path, req.Config)
	resp.Diagnostics.Append(diags...)
}
