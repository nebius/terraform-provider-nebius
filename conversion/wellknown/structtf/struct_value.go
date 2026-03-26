package structtf

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
)

var (
	_ basetypes.DynamicValuableWithSemanticEquals = (*Struct)(nil)
	_ xattr.ValidateableAttribute                 = (*Struct)(nil)
)

type Struct struct {
	basetypes.DynamicValue
}

// ValidateAttribute implements xattr.ValidateableAttribute.
func (v *Struct) ValidateAttribute(
	ctx context.Context,
	req xattr.ValidateAttributeRequest,
	resp *xattr.ValidateAttributeResponse,
) {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return
	}
	_, _, diags := dynamicToStruct(ctx, v.DynamicValue)
	resp.Diagnostics.Append(diags...)
}

// DynamicSemanticEquals implements basetypes.DynamicValuableWithSemanticEquals.
func (v *Struct) DynamicSemanticEquals(ctx context.Context, o basetypes.DynamicValuable) (bool, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	other, ok := o.(Struct)

	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic "+
				"equality checks. Please report this to the provider "+
				"developers.\n\nExpected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", o),
		)
		return false, diags
	}
	vbp, vUnk, innerDiag := v.ValueStructpb(ctx)
	diags.Append(innerDiag...)
	opb, oUnk, innerDiag := other.ValueStructpb(ctx)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return false, diags
	}

	if vUnk != nil || oUnk != nil {
		return true, diags
	}

	return proto.Equal(vbp, opb), diags
}

// Type returns a StructType.
func (v Struct) Type(_ context.Context) attr.Type {
	return StructTypeType
}

// Equal returns true if the given value is equivalent.
func (v Struct) Equal(o attr.Value) bool {
	other, ok := o.(Struct)

	if !ok {
		return false
	}

	return v.DynamicValue.Equal(other.DynamicValue)
}

// ValueMessage creates a new proto.Message instance from this Struct.
func (v Struct) ValueMessage(ctx context.Context) (
	proto.Message, *mask.Mask, diag.Diagnostics,
) {
	return v.ValueStructpb(ctx)
}

func (v Struct) ValueStructpb(ctx context.Context) (
	*structpb.Struct, *mask.Mask, diag.Diagnostics,
) {
	return dynamicToStruct(ctx, v.DynamicValue)
}

func NewStructNull() Struct {
	return Struct{
		DynamicValue: basetypes.NewDynamicNull(),
	}
}

func NewStructUnknown() Struct {
	return Struct{
		DynamicValue: basetypes.NewDynamicUnknown(),
	}
}

func NewStructEmpty() Struct {
	return Struct{
		DynamicValue: basetypes.NewDynamicValue(
			types.ObjectValueMust(
				map[string]attr.Type{},
				map[string]attr.Value{},
			),
		),
	}
}

func dynamicToStruct(
	ctx context.Context, value basetypes.DynamicValuable,
) (*structpb.Struct, *mask.Mask, diag.Diagnostics) {
	val, _, diags := ctypes.UnwrapDynamic(ctx, value)
	if diags.HasError() {
		return nil, nil, diags
	}
	if val.IsUnknown() {
		return nil, mask.New(), diags
	}
	if val.IsNull() {
		return nil, nil, diags
	}
	obj, ok := val.(types.Object)
	if !ok {
		diags.AddError(
			"Struct Conversion Error",
			fmt.Sprintf(
				"Expected types.Object, got %T",
				val,
			),
		)
		return nil, nil, diags
	}
	objFields := obj.Attributes()
	fields := make(map[string]*structpb.Value, len(objFields))
	var unk *mask.Mask
	for k, v := range objFields {
		dyn := basetypes.NewDynamicValue(v)
		fieldValue, innerUnk, innerDiags := dynamicToValue(ctx, dyn)
		diags.Append(innerDiags...)
		if diags.HasError() {
			continue
		}
		unk = ctypes.AppendUnknownMask(unk, mask.FieldPath{mask.FieldKey(k)}, innerUnk)
		fields[k] = fieldValue
	}
	return &structpb.Struct{
		Fields: fields,
	}, unk, diags
}

func structToDynamic(ctx context.Context, value *structpb.Struct) (
	basetypes.DynamicValue, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if value == nil {
		return basetypes.NewDynamicNull(), diags
	}
	elements := make(map[string]attr.Value, len(value.GetFields()))
	types := make(map[string]attr.Type, len(value.GetFields()))
	for k, v := range value.GetFields() {
		dyn, innerDiags := valueToDynamic(ctx, v)
		diags.Append(innerDiags...)
		el, _, innerDiags := ctypes.UnwrapDynamic(ctx, dyn)
		diags.Append(innerDiags...)
		elements[k] = el
		types[k] = el.Type(ctx)
	}
	obj, innerDiag := basetypes.NewObjectValue(
		types,
		elements,
	)
	diags.Append(innerDiag...)
	return basetypes.NewDynamicValue(
		obj,
	), diags
}

func NewStructValue(ctx context.Context, value *structpb.Struct) (
	Struct, diag.Diagnostics,
) {
	dyn, diags := structToDynamic(ctx, value)
	if diags.HasError() {
		return Struct{}, diags
	}
	return Struct{
		DynamicValue: dyn,
	}, diags
}

func NewStructValueMust(ctx context.Context, value *structpb.Struct) Struct {
	ret, diag := NewStructValue(ctx, value)
	if diag.HasError() {
		panic(fmt.Sprintf(
			"NewStructValue failed: %v",
			diag,
		))
	}
	return ret
}

func NewMessageValueMust(ctx context.Context, value proto.Message) Struct {
	ret, diags := NewMessageValue(ctx, value)
	if diags.HasError() {
		panic(fmt.Sprintf(
			"NewMessageValue failed: %v",
			diags,
		))
	}
	return ret
}

// NewMessageValue creates a Struct from a message.
func NewMessageValue(ctx context.Context, value proto.Message) (
	Struct, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if value == nil {
		return NewStructNull(), diags
	}
	structValue, ok := value.(*structpb.Struct)
	if !ok {
		diags.AddError(
			"NewStructValue Error",
			fmt.Sprintf(
				"Expected *structpb.Struct, got %T",
				value,
			),
		)
		return NewStructNull(), diags
	}
	return NewStructValue(ctx, structValue)
}
