package conversion

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/anytf"
)

func init() {
	anytf.DynamicMessageFromTF = MessageFromDynamic //nolint:reassign // update global var to avoid cyclic imports
}

func unsupportedTypeConversionDiag(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) diag.Diagnostics {
	return DiagnosticsFromAttributeErrString(
		attrPath,
		"unsupported type conversion",
		fmt.Sprintf("unsupported type conversion from terraform %s to protobuf %s", from.Type(ctx), to.Kind()),
	)
}

func BoolValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.BoolValuable)
	if !ok {
		return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
	}
	boolValue, d := typedAttr.ToBoolValue(ctx)
	return protoreflect.ValueOf(boolValue.ValueBool()), d, !d.HasError()
}

func Int32ValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.Int64Valuable)
	var val int64
	if !ok {
		typedAttr, ok := from.(basetypes.NumberValuable)
		if !ok {
			return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
		}
		bfAttr, d := typedAttr.ToNumberValue(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		var acc big.Accuracy
		val, acc = bfAttr.ValueBigFloat().Int64()
		if acc != big.Exact {
			return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
				attrPath,
				"attribute wrong accuracy",
				fmt.Sprintf("attribute %s has wrong accuracy for %s", from.Type(ctx), to.Kind()),
			), false
		}
	} else {
		intVal, d := typedAttr.ToInt64Value(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val = intVal.ValueInt64()
	}
	if val < math.MinInt32 || val > math.MaxInt32 {
		return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
			attrPath,
			"attribute out of range",
			fmt.Sprintf("attribute %s out of range for %s", from.Type(ctx), to.Kind()),
		), false
	}
	return protoreflect.ValueOf(int32(val)), diag.Diagnostics{}, true
}

func Uint32ValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.Int64Valuable)
	var val int64
	if !ok {
		typedAttr, ok := from.(basetypes.NumberValuable)
		if !ok {
			return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
		}
		bfAttr, d := typedAttr.ToNumberValue(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		var acc big.Accuracy
		val, acc = bfAttr.ValueBigFloat().Int64()
		if acc != big.Exact {
			return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
				attrPath,
				"attribute wrong accuracy",
				fmt.Sprintf("attribute %s has wrong accuracy for %s", from.Type(ctx), to.Kind()),
			), false
		}
	} else {
		intVal, d := typedAttr.ToInt64Value(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val = intVal.ValueInt64()
	}
	if val < 0 || val > math.MaxUint32 {
		return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
			attrPath,
			"attribute out of range",
			fmt.Sprintf("attribute %s out of range for %s", from.Type(ctx), to.Kind()),
		), false
	}
	return protoreflect.ValueOf(uint32(val)), diag.Diagnostics{}, true
}

func Int64ValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.Int64Valuable)
	var val int64
	if !ok {
		typedAttr, ok := from.(basetypes.NumberValuable)
		if !ok {
			return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
		}
		bfAttr, d := typedAttr.ToNumberValue(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		var acc big.Accuracy
		val, acc = bfAttr.ValueBigFloat().Int64()
		if acc != big.Exact {
			return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
				attrPath,
				"attribute wrong accuracy",
				fmt.Sprintf("attribute %s has wrong accuracy for %s", from.Type(ctx), to.Kind()),
			), false
		}
	} else {
		intVal, d := typedAttr.ToInt64Value(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val = intVal.ValueInt64()
	}
	return protoreflect.ValueOf(val), diag.Diagnostics{}, true
}

func Uint64ValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.NumberValuable)
	if !ok {
		return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
	}
	bfAttr, d := typedAttr.ToNumberValue(ctx)
	if d.HasError() {
		return protoreflect.Value{}, d, false
	}
	res, acc := bfAttr.ValueBigFloat().Uint64()
	if acc != 0 {
		return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
			attrPath,
			"attribute out of range",
			fmt.Sprintf("attribute %s out of range for %s", from.Type(ctx), to.Kind()),
		), false
	}
	return protoreflect.ValueOf(res), diag.Diagnostics{}, true
}

func Float32ValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.Float64Valuable)
	var val float64
	if !ok {
		typedAttr, ok := from.(basetypes.NumberValuable)
		if !ok {
			return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
		}
		bfAttr, d := typedAttr.ToNumberValue(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val, _ = bfAttr.ValueBigFloat().Float64()
	} else {
		floatVal, d := typedAttr.ToFloat64Value(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val = floatVal.ValueFloat64()
	}
	return protoreflect.ValueOf(float32(val)), diag.Diagnostics{}, true
}

func Float64ValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.Float64Valuable)
	var val float64
	if !ok {
		typedAttr, ok := from.(basetypes.NumberValuable)
		if !ok {
			return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
		}
		bfAttr, d := typedAttr.ToNumberValue(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val, _ = bfAttr.ValueBigFloat().Float64()
	} else {
		floatVal, d := typedAttr.ToFloat64Value(ctx)
		if d.HasError() {
			return protoreflect.Value{}, d, false
		}
		val = floatVal.ValueFloat64()
	}
	return protoreflect.ValueOf(val), diag.Diagnostics{}, true
}

func StringValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	stringable, ok := from.(basetypes.StringValuable)
	if !ok {
		return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
	}
	stringValue, d := stringable.ToStringValue(ctx)
	if d.HasError() {
		return protoreflect.Value{}, d, false
	}
	return protoreflect.ValueOf(stringValue.ValueString()), diag.Diagnostics{}, true
}

func BytesValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, attrPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.StringValuable)
	if !ok {
		return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, attrPath), false
	}
	stringValue, d := typedAttr.ToStringValue(ctx)
	if d.HasError() {
		return protoreflect.Value{}, d, false
	}
	res, err := base64.StdEncoding.DecodeString(stringValue.ValueString())
	if err != nil {
		d.AddAttributeError(
			attrPath,
			"base64 decode error",
			fmt.Sprintf("string to bytes base64 decode error: %s", err),
		)
		return protoreflect.ValueOf(res), d, false
	}
	return protoreflect.ValueOf(res), d, true
}

func EnumValueFromTF(ctx context.Context, from attr.Value, to protoreflect.FieldDescriptor, enumPath path.Path) (protoreflect.Value, diag.Diagnostics, bool) {
	typedAttr, ok := from.(basetypes.StringValuable)
	if !ok {
		return protoreflect.Value{}, unsupportedTypeConversionDiag(ctx, from, to, enumPath), false
	}
	stringValue, d := typedAttr.ToStringValue(ctx)
	if d.HasError() {
		return protoreflect.Value{}, d, false
	}
	enumVal := to.Enum().Values().ByName(protoreflect.Name(stringValue.ValueString()))
	if enumVal == nil {
		vals := []string{}
		for i := range to.Enum().Values().Len() {
			val := to.Enum().Values().Get(i)
			vals = append(vals, string(val.Name()))
		}
		return protoreflect.Value{}, DiagnosticsFromAttributeErrString(
			enumPath,
			"unsupported value",
			fmt.Sprintf(
				"unsupported or unknown value: %q, "+
					"only known values %s are allowed",
				stringValue.ValueString(), strings.Join(vals, ", "),
			),
		), false
	}
	return protoreflect.ValueOf(enumVal.Number()), diag.Diagnostics{}, true
}

func messageValueFromTF(
	ctx context.Context,
	newField protoreflect.Value,
	from attr.Value,
	messagePath path.Path,
	nameMap map[string]map[string]string,
) (protoreflect.Value, diag.Diagnostics, *mask.Mask, bool, bool) {
	msg := newField.Message().Interface()
	if wt, ok := wellknown.WellKnownOf(newField.Message().Descriptor()); ok {
		_, isDynamic := from.(basetypes.DynamicValuable)
		if from.Type(ctx).Equal(wt.Type()) || isDynamic {
			var unk *mask.Mask
			if from.IsUnknown() {
				unk = ctypes.AppendUnknownPath(unk, mask.FieldPath{})
			}
			res, unkInner, d := wt.FromValue(ctx, from)
			unk = ctypes.AppendUnknownMask(unk, mask.FieldPath{}, unkInner)
			proto.Merge(msg, res)
			return newField, d, unk, !d.HasError(), true
		}
	}
	from, _, innerDiag := ctypes.UnwrapDynamic(ctx, from)
	if innerDiag.HasError() {
		return newField, innerDiag, nil, false, true
	}
	if obj, ok := from.(basetypes.ObjectValuable); ok {
		unknowns, diagnostics := MessageFromTFPath(ctx, obj, msg, messagePath, nameMap)
		return newField, diagnostics, unknowns, true, true
	}
	if val, ok := from.(jsontypes.Normalized); ok {
		if err := protojson.Unmarshal([]byte(val.ValueString()), msg); err != nil {
			return newField, DiagnosticsFromAttributeErrString(
				messagePath,
				"unmarshal failed",
				fmt.Sprintf(
					"unmarshal protojson into protomessage failed: %s",
					err.Error(),
				),
			), nil, false, true
		}
		return newField, diag.Diagnostics{}, nil, true, true
	}
	return newField, diag.Diagnostics{}, nil, false, false
}

func MessageValueFromTF(
	ctx context.Context,
	newField protoreflect.Value,
	from attr.Value,
	to protoreflect.FieldDescriptor,
	messagePath path.Path,
	nameMap map[string]map[string]string,
) (protoreflect.Value, diag.Diagnostics, *mask.Mask, bool) {
	n, d, u, s, f := messageValueFromTF(ctx, newField, from, messagePath, nameMap)
	if f {
		return n, d, u, s
	}
	return newField, unsupportedTypeConversionDiag(ctx, from, to, messagePath), nil, false
}

func ValueFromTFKind(
	ctx context.Context,
	newField protoreflect.Value,
	from attr.Value,
	to protoreflect.FieldDescriptor,
	valPath path.Path,
	nameMap map[string]map[string]string,
) (protoreflect.Value, diag.Diagnostics, *mask.Mask, bool) {
	unwrapped, _, diags := ctypes.UnwrapDynamic(ctx, from)
	if diags.HasError() {
		return protoreflect.Value{}, diags, nil, false
	}
	val := protoreflect.Value{}
	ok := false
	var unknowns *mask.Mask
	var diagnostics diag.Diagnostics
	switch to.Kind() {
	case protoreflect.BoolKind:
		val, diagnostics, ok = BoolValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		val, diagnostics, ok = Int32ValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		val, diagnostics, ok = Uint32ValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		val, diagnostics, ok = Int64ValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		val, diagnostics, ok = Uint64ValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.FloatKind:
		val, diagnostics, ok = Float32ValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.DoubleKind:
		val, diagnostics, ok = Float64ValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.StringKind:
		val, diagnostics, ok = StringValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.BytesKind:
		val, diagnostics, ok = BytesValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.EnumKind:
		val, diagnostics, ok = EnumValueFromTF(ctx, unwrapped, to, valPath)
	case protoreflect.MessageKind:
		val, diagnostics, unknowns, ok = MessageValueFromTF(ctx, newField, from, to, valPath, nameMap)
	default: // nocov // no easy testing for this condition, coverage skipped
		diagnostics = DiagnosticsFromAttributeErrString(
			valPath,
			"protobuf type not supported",
			fmt.Sprintf("terraform to protobuf type conversion not supported for %s", to.Kind()),
		)
	}
	return val, diagnostics, unknowns, ok
}

func ListFieldFromTF(
	ctx context.Context,
	parent protoreflect.Message,
	from attr.Value,
	to protoreflect.FieldDescriptor,
	listPath path.Path,
	nameMap map[string]map[string]string,
) (diag.Diagnostics, *mask.Mask) {
	var elements []attr.Value
	diagnostics := diag.Diagnostics{}
	listValuable, ok := from.(basetypes.ListValuable)
	if !ok {
		tupleValue, ok := from.(basetypes.TupleValue)
		if !ok {
			return DiagnosticsFromAttributeErrString(
				listPath,
				"unsupported type conversion",
				fmt.Sprintf("unsupported type conversion from terraform %s to protobuf []%s", from.Type(ctx), to.Kind()),
			), nil
		}
		elements = tupleValue.Elements()
	} else {
		listFrom, diagnostics := listValuable.ToListValue(ctx)
		if diagnostics.HasError() {
			return diagnostics, nil
		}
		elements = listFrom.Elements()
	}
	listField := parent.NewField(to)
	parent.Set(to, listField)
	listField = parent.Mutable(to)
	listTo := listField.List()
	var unknowns *mask.Mask
	hasUnknowns := false
	for i, attr := range elements {
		keyPath := listPath.AtListIndex(i)
		if attr.IsUnknown() || attr.IsNull() {
			if attr.IsUnknown() {
				hasUnknowns = true
			}
			continue
		}
		val, innerDiag, innerUnknowns, ok := ValueFromTFKind(
			ctx, listTo.NewElement(), attr, to, keyPath, nameMap,
		)
		unknowns = ctypes.AppendUnknownMask(
			unknowns,
			mask.FieldPath{mask.FieldKey(strconv.Itoa(i))},
			innerUnknowns,
		)
		diagnostics.Append(innerDiag...)
		if ok {
			listTo.Append(val)
		}
	}
	if hasUnknowns {
		unknowns = ctypes.AppendUnknownPath(unknowns, mask.FieldPath{})
	}
	return diagnostics, unknowns
}

// mapKeyUnmarshalDiagnostics generates diagnostics for a failed map key unmarshaling operation.
//
// Parameters:
//   - err: The error occurred during unmarshaling.
//   - from: The string representation of the map key.
//   - to: The protoreflect.FieldDescriptor representing the expected type of the map key.
//   - keyPath: path to the key that failed.
//
// Returns:
//   - diagnostics: A slice of diagnostics indicating the failure to unmarshal the map key.
func mapKeyUnmarshalDiagnostics(err error, from string, to protoreflect.FieldDescriptor, keyPath path.Path) diag.Diagnostics {
	return DiagnosticsFromAttributeErrString(
		keyPath,
		"failed to unmarshal map key",
		fmt.Sprintf("failed to unmarshal map key %q into %s: %s", from, to.Kind(), err.Error()),
	)
}

// MapKeyFromString converts a string key from Terraform map to protoreflect.MapKey.
//
// Parameters:
//   - from: The string key from Terraform map.
//   - to: The protoreflect.FieldDescriptor representing the expected type of the map key.
//   - keyPath: path to the map key converted
//
// Returns:
//   - ret: The protoreflect.MapKey resulting from the conversion.
//   - diagnostics: A slice of diagnostics indicating any errors during the conversion process.
func MapKeyFromString(from string, to protoreflect.FieldDescriptor, keyPath path.Path) (protoreflect.MapKey, diag.Diagnostics) {
	var ret protoreflect.MapKey
	//nolint:exhaustive // default covers all other cases here
	switch to.Kind() {
	case protoreflect.BoolKind:
		val, err := strconv.ParseBool(from)
		if err != nil {
			return ret, mapKeyUnmarshalDiagnostics(err, from, to, keyPath)
		}
		ret = protoreflect.ValueOf(val).MapKey()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		val, err := strconv.ParseInt(from, 10, 32)
		if err != nil {
			return ret, mapKeyUnmarshalDiagnostics(err, from, to, keyPath)
		}
		ret = protoreflect.ValueOf(int32(val)).MapKey()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		val, err := strconv.ParseUint(from, 10, 32)
		if err != nil {
			return ret, mapKeyUnmarshalDiagnostics(err, from, to, keyPath)
		}
		ret = protoreflect.ValueOf(uint32(val)).MapKey()
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		val, err := strconv.ParseInt(from, 10, 64)
		if err != nil {
			return ret, mapKeyUnmarshalDiagnostics(err, from, to, keyPath)
		}
		ret = protoreflect.ValueOf(val).MapKey()
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		val, err := strconv.ParseUint(from, 10, 64)
		if err != nil {
			return ret, mapKeyUnmarshalDiagnostics(err, from, to, keyPath)
		}
		ret = protoreflect.ValueOf(val).MapKey()
	case protoreflect.StringKind:
		ret = protoreflect.ValueOf(from).MapKey()
	default:
		return ret, DiagnosticsFromAttributeErrString(
			keyPath,
			"mapkey type not supported",
			fmt.Sprintf("terraform to protobuf type conversion not supported for %s", to.Kind()),
		)
	}
	return ret, diag.Diagnostics{}
}

// MapFieldFromTF converts Terraform map to a protoreflect.FieldDescriptor of type map.
// It maps values from a types.Map (representing Terraform map attribute) to a map protoreflect.FieldDescriptor
// in a protoreflect.Message.
// It returns diag.Diagnostics and ctypes.Unknowns to handle unknown or erroneous attributes.
//
// Parameters:
//   - ctx: The context.Context.
//   - parent: The protoreflect.Message containing the map field.
//   - from: The attr.Value representing Terraform map attributes.
//   - to: The protoreflect.FieldDescriptor representing the target map field.
//   - mapPath: path to the map being converted
//   - nameMap: A map of custom names of protobuf fields with the following structure:
//     field.parent.full_name -> terraform field name -> proto field name
//
// Returns:
//   - diagnostics: A slice of diagnostics indicating any errors or issues during the conversion process.
//   - unknowns: ctypes.Unknowns containing information about attributes that are unknown at the time of execution.
func MapFieldFromTF(
	ctx context.Context,
	parent protoreflect.Message,
	from attr.Value,
	to protoreflect.FieldDescriptor,
	mapPath path.Path,
	nameMap map[string]map[string]string,
) (diag.Diagnostics, *mask.Mask) {
	var elements map[string]attr.Value
	diagnostics := diag.Diagnostics{}
	mapValuable, ok := from.(basetypes.MapValuable)
	if !ok {
		objValuable, ok := from.(basetypes.ObjectValuable)
		if !ok {
			return DiagnosticsFromAttributeErrString(
				mapPath,
				"unsupported type conversion",
				fmt.Sprintf("unsupported type conversion from terraform %s to protobuf map[%s]%s", from.Type(ctx), to.MapKey().Kind(), to.MapValue().Kind()),
			), nil
		}
		objFrom, diagnostics := objValuable.ToObjectValue(ctx)
		if diagnostics.HasError() {
			return diagnostics, nil
		}
		elements = objFrom.Attributes()
	} else {
		mapFrom, diagnostics := mapValuable.ToMapValue(ctx)
		if diagnostics.HasError() {
			return diagnostics, nil
		}
		elements = mapFrom.Elements()
	}
	mapField := parent.NewField(to)
	parent.Set(to, mapField)
	mapField = parent.Mutable(to)
	mapTo := mapField.Map()
	var unknowns *mask.Mask
	hasUnknowns := false
	for key, attr := range elements {
		keyPath := mapPath.AtMapKey(key)
		protoKey, innerDiag := MapKeyFromString(key, to.MapKey(), keyPath)
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() {
			continue
		}

		if attr.IsUnknown() || attr.IsNull() {
			if attr.IsUnknown() {
				hasUnknowns = true
			}
			continue
		}
		val, innerDiag, innerUnknowns, ok := ValueFromTFKind(
			ctx, mapTo.NewValue(), attr, to.MapValue(), keyPath, nameMap,
		)
		diagnostics.Append(innerDiag...)
		unknowns = ctypes.AppendUnknownMask(
			unknowns,
			mask.FieldPath{mask.FieldKey(fmt.Sprint(protoKey.Interface()))},
			innerUnknowns,
		)
		if ok {
			mapTo.Set(protoKey, val)
		}
	}
	if hasUnknowns {
		unknowns = ctypes.AppendUnknownPath(unknowns, mask.FieldPath{})
	}
	return diagnostics, unknowns
}

// fieldNameByTFNameAndParent returns the protobuf field name that corresponds
// to tfName under the given parent message, computing the per‑parent map once.
func fieldNameByTFNameAndParent(
	tfName string,
	parent protoreflect.MessageDescriptor,
	nameMap map[string]map[string]string,
) (protoreflect.Name, error) {
	if nameMap == nil {
		return protoreflect.Name(tfName), nil
	}
	if fieldsOfParent, ok := nameMap[string(parent.FullName())]; ok {
		if n, hit := fieldsOfParent[tfName]; hit {
			return protoreflect.Name(n), nil
		}
	}
	return protoreflect.Name(tfName), nil
}

// MessageFromTFPath converts Terraform attributes to a protobuf message
// recursively. It maps values from a [types.Object] (representing Terraform
// attributes) to a [proto.Message] based on their respective protobuf message
// descriptors. The function supports nested structures, lists, and maps. It
// returns [ctypes.Unknowns] and [diag.Diagnostics] with all the conversion errors.
//
// Values that are not present in the message are being skipped.
//
// Parameters:
//   - ctx: [context.Context].
//   - from: [types.Object] representing Terraform attributes.
//   - to: [proto.Message] to which attributes are mapped.
//   - messagePath: path to the converted message.
//   - nameMap: map of custom names of protobuf fields with the following structure:
//     field.parent.full_name -> terraform field name -> proto field name
//
// Returns:
//   - unknowns: [ctypes.Unknowns] containing attribute paths that were not known at
//     the time of the execution.
//   - diagnostics: [diag.Diagnostics] indicating any errors or issues
//     during the conversion process.
func MessageFromTFPath(
	ctx context.Context,
	from basetypes.ObjectValuable,
	to proto.Message,
	messagePath path.Path,
	nameMap map[string]map[string]string,
) (*mask.Mask, diag.Diagnostics) {
	if from.IsUnknown() || from.IsNull() {
		return nil, diag.Diagnostics{}
	}
	fromObj, diagnostics := from.ToObjectValue(ctx)

	var unknowns *mask.Mask

	attrs := fromObj.Attributes()
	toReflect := to.ProtoReflect()
	toDesc := toReflect.Descriptor()

	for key, attr := range attrs {
		attrPath := messagePath.AtName(key)
		pbName, err := fieldNameByTFNameAndParent(key, toDesc, nameMap)
		if err != nil {
			diagnostics.AddAttributeError(
				attrPath,
				"get field name error",
				fmt.Sprintf("get field name for %q in %s: %s", key, toDesc.FullName(), err.Error()),
			)
			continue
		}
		protoFieldDesc := toDesc.Fields().ByName(pbName)
		if protoFieldDesc == nil || attr.IsNull() {
			continue
		}
		unwrappedAttr, _, unwrapDiag := ctypes.UnwrapDynamic(ctx, attr)
		diagnostics.Append(unwrapDiag...)
		if unwrapDiag.HasError() {
			continue
		}
		if unwrappedAttr.IsUnknown() {
			if protoFieldDesc.ContainingOneof() != nil {
				unknowns = ctypes.AppendUnknownPath(
					unknowns,
					mask.FieldPath{
						mask.FieldKey(string(protoFieldDesc.ContainingOneof().Name())),
					},
				)
			}
			unknowns = ctypes.AppendUnknownPath(
				unknowns,
				mask.FieldPath{mask.FieldKey(string(pbName))},
			)
			continue
		}
		if protoFieldDesc.IsList() {
			innerDiag, innerUnknowns := ListFieldFromTF(
				ctx, toReflect, unwrappedAttr, protoFieldDesc, attrPath,
				nameMap,
			)
			unknowns = ctypes.AppendUnknownMask(
				unknowns,
				mask.FieldPath{mask.FieldKey(string(pbName))},
				innerUnknowns,
			)
			diagnostics.Append(innerDiag...)
			continue
		}
		if protoFieldDesc.IsMap() {
			innerDiag, innerUnknowns := MapFieldFromTF(
				ctx, toReflect, unwrappedAttr, protoFieldDesc, attrPath,
				nameMap,
			)
			unknowns = ctypes.AppendUnknownMask(
				unknowns,
				mask.FieldPath{mask.FieldKey(string(pbName))},
				innerUnknowns,
			)
			diagnostics.Append(innerDiag...)
			continue
		}
		val, innerDiag, innerUnknowns, ok := ValueFromTFKind(
			ctx, toReflect.NewField(protoFieldDesc), attr, protoFieldDesc,
			attrPath, nameMap,
		)
		unknowns = ctypes.AppendUnknownMask(
			unknowns,
			mask.FieldPath{mask.FieldKey(string(pbName))},
			innerUnknowns,
		)
		diagnostics.Append(innerDiag...)
		if ok {
			toReflect.Set(protoFieldDesc, val)
		}
	}
	return unknowns, diagnostics
}

func MessageFromDynamicRecursive(
	ctx context.Context,
	from basetypes.DynamicValuable,
	to proto.Message,
	messagePath path.Path,
	nameMap map[string]map[string]string,
) (*mask.Mask, diag.Diagnostics) {
	unwrapped, _, diags := ctypes.UnwrapDynamic(ctx, from)
	if diags.HasError() {
		return nil, diags
	}
	if unwrapped.IsUnknown() || unwrapped.IsNull() {
		return nil, diags
	}
	_, innerDiag, innerUnknowns, _, found := messageValueFromTF(
		ctx, protoreflect.ValueOfMessage(to.ProtoReflect()), from, messagePath,
		nameMap,
	)
	if !found {
		innerDiag.AddAttributeError(
			messagePath,
			"unsupported dynamic message conversion",
			fmt.Sprintf(
				"unsupported dynamic message conversion from terraform %s"+
					" to protobuf %s",
				from.Type(ctx), to.ProtoReflect().Descriptor().FullName(),
			),
		)
	}
	return innerUnknowns, innerDiag
}

// MessageFromTF converts Terraform attributes to a protobuf message.
// It maps values from a [types.Object] (representing Terraform attributes) to
// a [proto.Message] based on their respective protobuf message descriptors. The
// function supports nested structures, lists, and maps. It returns [ctypes.Unknowns]
// and [diag.Diagnostics] with all the conversion errors.
// It is an entry call for [MessageFromTFPath](ctx, from, to, path), where
// path is "" as this is the root message.
//
// Parameters:
//   - ctx: [context.Context].
//   - from: [types.Object] representing Terraform attributes.
//   - to: [proto.Message] to which attributes are mapped.
//   - nameMap: map of custom names of protobuf fields with the following structure:
//     field.parent.full_name -> terraform field name -> proto field name
//
// Returns:
//   - unknowns: [ctypes.Unknowns] containing attribute paths that were not known at
//     the time of the execution.
//   - diagnostics: [diag.Diagnostics] A slice of diagnostics indicating any
//     errors or issues during the conversion process.
func MessageFromTF(
	ctx context.Context,
	from basetypes.ObjectValuable,
	to proto.Message,
	nameMap map[string]map[string]string,
) (
	*mask.Mask, diag.Diagnostics,
) {
	return MessageFromTFPath(ctx, from, to, path.Empty(), nameMap)
}

func MessageFromDynamic(
	ctx context.Context,
	from basetypes.DynamicValuable,
	to proto.Message,
	nameMap map[string]map[string]string,
) (*mask.Mask, diag.Diagnostics) {
	return MessageFromDynamicRecursive(ctx, from, to, path.Empty(), nameMap)
}
