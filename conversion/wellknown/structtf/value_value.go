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
	_ basetypes.DynamicValuableWithSemanticEquals = (*Value)(nil)
	_ xattr.ValidateableAttribute                 = (*Value)(nil)
)

type Value struct {
	basetypes.DynamicValue
}

// ValidateAttribute implements xattr.ValidateableAttribute.
func (v *Value) ValidateAttribute(
	ctx context.Context,
	req xattr.ValidateAttributeRequest,
	resp *xattr.ValidateAttributeResponse,
) {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return
	}
	_, _, diags := dynamicToValue(ctx, v.DynamicValue)
	resp.Diagnostics.Append(diags...)
}

// DynamicSemanticEquals implements basetypes.DynamicValuableWithSemanticEquals.
func (v *Value) DynamicSemanticEquals(ctx context.Context, o basetypes.DynamicValuable) (bool, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	other, ok := o.(Value)

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

// Type returns a ValueType.
func (v Value) Type(_ context.Context) attr.Type {
	return ValueTypeType
}

// Equal returns true if the given value is equivalent.
func (v Value) Equal(o attr.Value) bool {
	other, ok := o.(Value)

	if !ok {
		return false
	}

	return v.DynamicValue.Equal(other.DynamicValue)
}

// ValueMessage creates a new proto.Message instance from this Value.
func (v Value) ValueMessage(ctx context.Context) (
	proto.Message, *mask.Mask, diag.Diagnostics,
) {
	return v.ValueStructpb(ctx)
}

func (v Value) ValueStructpb(ctx context.Context) (
	*structpb.Value, *mask.Mask, diag.Diagnostics,
) {
	return dynamicToValue(ctx, v.DynamicValue)
}

func NewValueNull() Value {
	return Value{
		DynamicValue: basetypes.NewDynamicNull(),
	}
}

func NewValueUnknown() Value {
	return Value{
		DynamicValue: basetypes.NewDynamicUnknown(),
	}
}

func NewValueEmpty() Value {
	return Value{
		DynamicValue: basetypes.NewDynamicNull(),
	}
}

func dynamicToValue(ctx context.Context, dyn basetypes.DynamicValuable) (
	*structpb.Value, *mask.Mask, diag.Diagnostics,
) {
	var unk *mask.Mask
	val, _, diags := ctypes.UnwrapDynamic(ctx, dyn)
	if diags.HasError() {
		return nil, nil, diags
	}
	if val.IsNull() {
		return structpb.NewNullValue(), nil, diags
	}
	if val.IsUnknown() {
		return structpb.NewNullValue(), mask.New(), diags
	}
	switch v := val.(type) {
	case types.String:
		return structpb.NewStringValue(v.ValueString()), nil, diags
	case types.Number:
		floatNum, _ := v.ValueBigFloat().Float64()
		return structpb.NewNumberValue(floatNum), nil, diags
	case types.Int32:
		return structpb.NewNumberValue(float64(v.ValueInt32())), nil, diags
	case types.Int64:
		return structpb.NewNumberValue(float64(v.ValueInt64())), nil, diags
	case types.Float32:
		return structpb.NewNumberValue(float64(v.ValueFloat32())), nil, diags
	case types.Float64:
		return structpb.NewNumberValue(v.ValueFloat64()), nil, diags
	case types.Bool:
		return structpb.NewBoolValue(v.ValueBool()), nil, diags
	case types.Object:
		dyn := basetypes.NewDynamicValue(v)
		structValue, innerUnk, innerDiags := dynamicToStruct(ctx, dyn)
		diags.Append(innerDiags...)
		if diags.HasError() {
			return nil, innerUnk, diags
		}
		unk = ctypes.AppendUnknownMask(unk, mask.FieldPath{}, innerUnk)
		if structValue == nil {
			return structpb.NewNullValue(), unk, diags
		}
		return structpb.NewStructValue(structValue), unk, diags
	case types.List:
		elements := make([]*structpb.Value, len(v.Elements()))
		for i, elem := range v.Elements() {
			dyn := basetypes.NewDynamicValue(elem)
			fieldValue, innerUnk, innerDiags := dynamicToValue(ctx, dyn)
			diags.Append(innerDiags...)
			if diags.HasError() {
				continue
			}
			unk = ctypes.AppendUnknownMask(
				unk,
				mask.FieldPath{mask.FieldKey(fmt.Sprintf("%d", i))},
				innerUnk,
			)
			elements[i] = fieldValue
		}
		return structpb.NewListValue(&structpb.ListValue{
			Values: elements,
		}), unk, diags
	case types.Map:
		fields := make(map[string]*structpb.Value, len(v.Elements()))
		for k, elem := range v.Elements() {
			dyn := basetypes.NewDynamicValue(elem)
			fieldValue, innerUnk, innerDiags := dynamicToValue(ctx, dyn)
			diags.Append(innerDiags...)
			if diags.HasError() {
				continue
			}
			unk = ctypes.AppendUnknownMask(
				unk,
				mask.FieldPath{mask.FieldKey(k)},
				innerUnk,
			)
			fields[k] = fieldValue
		}
		return structpb.NewStructValue(&structpb.Struct{
			Fields: fields,
		}), unk, diags
	case types.Tuple:
		elements := make([]*structpb.Value, len(v.Elements()))
		for i, elem := range v.Elements() {
			dyn := basetypes.NewDynamicValue(elem)
			fieldValue, innerUnk, innerDiags := dynamicToValue(ctx, dyn)
			diags.Append(innerDiags...)
			if diags.HasError() {
				continue
			}
			unk = ctypes.AppendUnknownMask(
				unk,
				mask.FieldPath{mask.FieldKey(fmt.Sprintf("%d", i))},
				innerUnk,
			)
			elements[i] = fieldValue
		}
		return structpb.NewListValue(&structpb.ListValue{
			Values: elements,
		}), unk, diags
	default:
		diags.AddError(
			"Dynamic To Value Error",
			fmt.Sprintf("Unsupported dynamic value type: %T", v),
		)
		return nil, nil, diags
	}
}

func valueToDynamic(ctx context.Context, value *structpb.Value) (
	basetypes.DynamicValue, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if value == nil {
		return basetypes.NewDynamicNull(), diags
	}
	switch v := value.GetKind().(type) {
	case nil:
		return basetypes.NewDynamicNull(), diags
	case *structpb.Value_NullValue:
		return basetypes.NewDynamicNull(), diags
	case *structpb.Value_NumberValue:
		return basetypes.NewDynamicValue(
			types.Float64Value(v.NumberValue),
		), diags
	case *structpb.Value_StringValue:
		return basetypes.NewDynamicValue(
			types.StringValue(v.StringValue),
		), diags
	case *structpb.Value_BoolValue:
		return basetypes.NewDynamicValue(
			types.BoolValue(v.BoolValue),
		), diags
	case *structpb.Value_StructValue:
		return structToDynamic(ctx, v.StructValue)
	case *structpb.Value_ListValue:
		list := v.ListValue
		if list == nil {
			return basetypes.NewDynamicValue(
				types.ListValueMust(
					types.ListType{
						ElemType: types.DynamicType,
					},
					[]attr.Value{},
				),
			), diags
		}
		elements := make([]attr.Value, len(list.GetValues()))
		elementTypes := make([]attr.Type, len(list.GetValues()))
		for i, v := range list.GetValues() {
			elemDyn, innerDiags := valueToDynamic(ctx, v)
			diags.Append(innerDiags...)
			elem, _, innerDiags := ctypes.UnwrapDynamic(ctx, elemDyn)
			diags.Append(innerDiags...)
			elements[i] = elem
			elementTypes[i] = elem.Type(ctx)
		}
		listVal, innerDiag := basetypes.NewTupleValue(
			elementTypes,
			elements,
		)
		diags.Append(innerDiag...)
		if diags.HasError() {
			return basetypes.DynamicValue{}, diags
		}
		return basetypes.NewDynamicValue(listVal), diags
	default:
		return basetypes.DynamicValue{}, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Unsupported Value Type",
				fmt.Sprintf("Unsupported value type: %T", v),
			),
		}
	}
}

func NewValue(ctx context.Context, value *structpb.Value) (
	Value, diag.Diagnostics,
) {
	dyn, diags := valueToDynamic(ctx, value)
	if diags.HasError() {
		return Value{}, diags
	}
	return Value{
		DynamicValue: dyn,
	}, diags
}

func NewValueMust(ctx context.Context, value *structpb.Value) Value {
	val, diags := NewValue(ctx, value)
	if diags.HasError() {
		panic(fmt.Sprintf(
			"NewValueMust failed: %v",
			diags,
		))
	}
	return val
}
