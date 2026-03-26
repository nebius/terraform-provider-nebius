package anytf

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/tftype"
)

var (
	_ basetypes.ObjectTypable = (*AnyType)(nil)
)

type AnyType struct {
	basetypes.ObjectType
}

var AnyTypeType = AnyType{
	ObjectType: basetypes.ObjectType{AttrTypes: anyAttrTypes},
}

func (t AnyType) TFType() tftype.TFType {
	return tftype.TFObject
}

func (t AnyType) Documentation() string {
	return "It's a wrapper around google.protobuf.Any that tries to convert " +
		"it to dynamic, if the message is known. Otherwise it will store it in b64."
}

func (t AnyType) String() string {
	return "any.AnyType"
}

func (t AnyType) Equal(o attr.Type) bool {
	other, ok := o.(AnyType)

	if !ok {
		return false
	}

	return t.ObjectType.Equal(other.ObjectType)
}

func (t AnyType) Type() attr.Type {
	return t
}

func (t AnyType) FromValue(
	ctx context.Context, val attr.Value,
) (proto.Message, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	valTyped, ok := val.(Any)
	if !ok {
		unwrapped, _, unwrapDiag := ctypes.UnwrapDynamic(ctx, val)
		diags.Append(unwrapDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		if unwrapped.IsNull() || unwrapped.IsUnknown() {
			return (*anypb.Any)(nil), nil, diags
		}
		objValuable, ok := unwrapped.(basetypes.ObjectValuable)
		if !ok {
			diags.AddError(
				"value is not "+t.String(),
				fmt.Sprintf(
					"value has to be %s, %q found",
					t.String(), val.Type(ctx).String(),
				),
			)
			return nil, nil, diags
		}
		obj, innerDiag := objValuable.ToObjectValue(ctx)
		diags.Append(innerDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		attrs := obj.Attributes()
		if len(attrs) > 2 {
			diags.AddError(
				"too many attributes for Any type",
				fmt.Sprintf(
					"expected only 'type_url' and 'value' attributes, got %d",
					len(attrs),
				),
			)
			return nil, nil, diags
		}
		typeURLAttr, ok := attrs["type_url"]
		if !ok {
			diags.AddError(
				"'type_url' attribute is not present",
				"'type_url' attribute has to be present in Any type",
			)
			return nil, nil, diags
		}
		typeURL, ok := typeURLAttr.(basetypes.StringValuable)
		if !ok {
			diags.AddError(
				"'type_url' attribute is not string",
				fmt.Sprintf(
					"'type_url' attribute has to be string, %T found",
					typeURLAttr,
				),
			)
			return nil, nil, diags
		}
		typeURL, innerDiag = typeURL.ToStringValue(ctx)
		diags.Append(innerDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		valueAttr, ok := attrs["value"]
		if !ok {
			diags.AddError(
				"'value' attribute is not present",
				"'value' attribute has to be present in Any type",
			)
			return nil, nil, diags
		}
		switch valAttrTyped := valueAttr.(type) {
		case basetypes.StringValuable:
			valueAttrStr, innerDiag := valAttrTyped.ToStringValue(ctx)
			diags.Append(innerDiag...)
			if diags.HasError() {
				return nil, nil, diags
			}
			valueAttr = basetypes.NewDynamicValue(valueAttrStr)
		case basetypes.ObjectValuable:
			valAttrObj, innerDiag := valAttrTyped.ToObjectValue(ctx)
			diags.Append(innerDiag...)
			if diags.HasError() {
				return nil, nil, diags
			}
			valueAttr = basetypes.NewDynamicValue(valAttrObj)
		default:
			diags.AddError(
				"'value' attribute has unsupported type",
				fmt.Sprintf(
					"'value' attribute has to be string or object, %T found",
					valueAttr,
				),
			)
			return nil, nil, diags
		}
		anyObjectValue, innerDiag := basetypes.NewObjectValue(
			t.AttrTypes,
			map[string]attr.Value{
				"type_url": typeURL,
				"value":    valueAttr,
			},
		)
		diags.Append(innerDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		valTyped = Any{
			ObjectValue: anyObjectValue,
		}
	}
	if valTyped.IsNull() || valTyped.IsUnknown() {
		return (*anypb.Any)(nil), nil, diag.Diagnostics{}
	}
	ret, unk, innerDiag := valTyped.ValueAnypb(ctx, nil)
	diags = append(diags, innerDiag...)
	return ret, unk, diags
}
func (t AnyType) ToValue(ctx context.Context, msg proto.Message) (
	attr.Value, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return t.Null(), diags
	}
	msgAny, ok := msg.(*anypb.Any)
	if !ok {
		diags.AddError(
			"message is not *anypb.Any",
			fmt.Sprintf(
				"message has to be *anypb.Any, %T found",
				msg,
			),
		)
		return nil, diags
	}
	ret, innerDiag := NewAnypbValue(ctx, msgAny)
	diags.Append(innerDiag...)
	return ret, diags
}
func (t AnyType) ToDynamicValue(
	ctx context.Context,
	msg proto.Message,
) (basetypes.DynamicValue, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return basetypes.NewDynamicNull(), diags
	}
	d, diags := t.ToValue(ctx, msg)
	if diags.HasError() {
		return basetypes.NewDynamicNull(), diags
	}
	ret, innerDiag := d.(Any).ToObjectValue(ctx)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return basetypes.NewDynamicNull(), diags
	}
	attrs := ret.Attributes()
	valUnwraped, _, unwrapDiag := ctypes.UnwrapDynamic(ctx, attrs["value"])
	diags.Append(unwrapDiag...)
	if diags.HasError() {
		return basetypes.NewDynamicNull(), diags
	}
	retAttrs := map[string]attr.Value{
		"type_url": attrs["type_url"],
		"value":    valUnwraped,
	}
	retTypes := map[string]attr.Type{
		"type_url": basetypes.StringType{},
		"value":    attrs["value"].Type(ctx),
	}
	objVal, innerDiag := basetypes.NewObjectValue(retTypes, retAttrs)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return basetypes.NewDynamicNull(), diags
	}
	return basetypes.NewDynamicValue(objVal), diags
}

func (t AnyType) Null() attr.Value {
	return NewAnyNull()
}
func (t AnyType) Unknown() attr.Value {
	return NewAnyUnknown()
}
func (t AnyType) Message() proto.Message {
	return &anypb.Any{}
}

func (t AnyType) Empty() attr.Value {
	return NewAnyEmpty()
}

// ValueFromObject returns a ObjectValuable type given a ObjectValue.
func (t AnyType) ValueFromObject(
	ctx context.Context,
	in basetypes.ObjectValue,
) (basetypes.ObjectValuable, diag.Diagnostics) {
	return Any{
		ObjectValue: in,
	}, nil
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to
// convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t AnyType) ValueFromTerraform(
	ctx context.Context,
	in tftypes.Value,
) (attr.Value, error) {
	attrValue, err := t.ObjectType.ValueFromTerraform(ctx, in)

	if err != nil {
		return nil, err
	}

	objValue, ok := attrValue.(basetypes.ObjectValue)

	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	objValuable, diags := t.ValueFromObject(ctx, objValue)

	if diags.HasError() {
		return nil, fmt.Errorf(
			"unexpected error converting ObjectValue to ObjectValuable: %v",
			diags,
		)
	}

	return objValuable, nil
}
