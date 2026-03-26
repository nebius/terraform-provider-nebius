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
	_ basetypes.DynamicTypable = (*StructType)(nil)
)

type StructType struct {
	basetypes.DynamicType
}

var StructTypeType = StructType{}

func (t StructType) TFType() tftype.TFType {
	return tftype.TFDynamic
}

func (t StructType) Documentation() string {
	return "It's a wrapper around google.protobuf.Struct that converts " +
		"it to dynamic."
}

func (t StructType) String() string {
	return "structtf.StructType"
}

func (t StructType) Equal(o attr.Type) bool {
	other, ok := o.(StructType)

	if !ok {
		return false
	}

	return t.DynamicType.Equal(other.DynamicType)
}

func (t StructType) Type() attr.Type {
	return t
}

func (t StructType) FromValue(
	ctx context.Context, val attr.Value,
) (proto.Message, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	obj, ok := val.(Struct)
	if !ok {
		unwrapped, _, unwrapDiag := ctypes.UnwrapDynamic(ctx, val)
		diags.Append(unwrapDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		if unwrapped.IsNull() || unwrapped.IsUnknown() {
			return (*structpb.Struct)(nil), nil, diags
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
		objValue, innerDiag := objValuable.ToObjectValue(ctx)
		diags = append(diags, innerDiag...)
		if innerDiag.HasError() {
			return nil, nil, diags
		}
		obj = Struct{
			DynamicValue: basetypes.NewDynamicValue(objValue),
		}
	}
	if obj.IsNull() || obj.IsUnknown() {
		return (*structpb.Struct)(nil), nil, diag.Diagnostics{}
	}
	ret, unk, innerDiag := obj.ValueStructpb(ctx)
	diags = append(diags, innerDiag...)
	return ret, unk, diags
}
func (t StructType) ToValue(ctx context.Context, msg proto.Message) (
	attr.Value, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return t.Null(), diags
	}
	msgStruct, ok := msg.(*structpb.Struct)
	if !ok {
		diags.AddError(
			"message is not *structpb.Struct",
			fmt.Sprintf(
				"message has to be *structpb.Struct, %T found",
				msg,
			),
		)
		return nil, diags
	}
	ret, innerDiag := NewStructValue(ctx, msgStruct)
	diags.Append(innerDiag...)
	return ret, diags
}
func (t StructType) ToDynamicValue(
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
	ret, innerDiag := d.(Struct).ToDynamicValue(ctx)
	diags.Append(innerDiag...)
	return ret, diags
}
func (t StructType) Null() attr.Value {
	return NewStructNull()
}
func (t StructType) Unknown() attr.Value {
	return NewStructUnknown()
}
func (t StructType) Message() proto.Message {
	return &structpb.Struct{}
}

func (t StructType) Empty() attr.Value {
	return NewStructEmpty()
}

func (t StructType) ValueFromDynamic(
	ctx context.Context,
	val basetypes.DynamicValue,
) (basetypes.DynamicValuable, diag.Diagnostics) {
	diags := diag.Diagnostics{}

	if val.IsNull() {
		return NewStructNull(), diags
	}

	if val.IsUnknown() {
		return NewStructUnknown(), diags
	}

	return Struct{
		DynamicValue: val,
	}, diags
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to
// convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t StructType) ValueFromTerraform(
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
