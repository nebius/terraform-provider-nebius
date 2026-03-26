package structtf

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/tftype"
)

var (
	_ basetypes.DynamicTypable = (*ValueType)(nil)
)

type ValueType struct {
	basetypes.DynamicType
}

var ValueTypeType = ValueType{}

func (t ValueType) TFType() tftype.TFType {
	return tftype.TFDynamic
}

func (t ValueType) Documentation() string {
	return "It's a wrapper around google.protobuf.Value that converts " +
		"it to dynamic."
}

func (t ValueType) String() string {
	return "structtf.ValueType"
}

func (t ValueType) Equal(o attr.Type) bool {
	other, ok := o.(ValueType)

	if !ok {
		return false
	}

	return t.DynamicType.Equal(other.DynamicType)
}

func (t ValueType) Type() attr.Type {
	return t
}

func (t ValueType) FromValue(
	ctx context.Context, val attr.Value,
) (proto.Message, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	obj, ok := val.(Value)
	if !ok {
		unwrapped, _, unwrapDiag := ctypes.UnwrapDynamic(ctx, val)
		diags = append(diags, unwrapDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		if unwrapped.IsNull() || unwrapped.IsUnknown() {
			return (*structpb.Value)(nil), nil, diags
		}
		obj = Value{
			DynamicValue: basetypes.NewDynamicValue(unwrapped),
		}
	}
	if obj.IsNull() || obj.IsUnknown() {
		return (*structpb.Value)(nil), nil, diag.Diagnostics{}
	}
	ret, unk, innerDiag := obj.ValueStructpb(ctx)
	diags = append(diags, innerDiag...)
	return ret, unk, diags
}
func (t ValueType) ToValue(ctx context.Context, msg proto.Message) (
	attr.Value, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return t.Null(), diags
	}
	msgStruct, ok := msg.(*structpb.Value)
	if !ok {
		diags.AddError(
			"message is not *structpb.Value",
			fmt.Sprintf(
				"message has to be *structpb.Value, %T found",
				msg,
			),
		)
		return nil, diags
	}
	ret, innerDiag := NewValue(ctx, msgStruct)
	diags.Append(innerDiag...)
	return ret, diags
}
func (t ValueType) ToDynamicValue(
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
	ret, innerDiag := d.(Value).ToDynamicValue(ctx)
	diags.Append(innerDiag...)
	return ret, diags
}

func (t ValueType) Null() attr.Value {
	return NewValueNull()
}
func (t ValueType) Unknown() attr.Value {
	return NewValueUnknown()
}
func (t ValueType) Message() proto.Message {
	return &structpb.Value{}
}

func (t ValueType) Empty() attr.Value {
	return NewValueEmpty()
}

func (t ValueType) ValueFromDynamic(
	ctx context.Context,
	val basetypes.DynamicValue,
) (basetypes.DynamicValuable, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if val.IsNull() {
		return NewValueNull(), diags
	}

	if val.IsUnknown() {
		return NewValueUnknown(), diags
	}

	return Value{
		DynamicValue: val,
	}, diags
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to
// convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t ValueType) ValueFromTerraform(
	ctx context.Context,
	in tftypes.Value,
) (attr.Value, error) {
	attrValue, err := t.DynamicType.ValueFromTerraform(ctx, in)

	if err != nil {
		return nil, err
	}

	val, ok := attrValue.(basetypes.DynamicValue)

	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	valuable, diags := t.ValueFromDynamic(ctx, val)

	if diags.HasError() {
		return nil, fmt.Errorf(
			"unexpected error converting DynamicValue to DynamicValuable: %v",
			diags,
		)
	}

	return valuable, nil
}
