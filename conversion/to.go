package conversion

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/nebius/gosdk/proto/nebius"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/anytf"
)

func init() {
	anytf.DynamicMessageToTF = DynamicMessageToTF //nolint:reassign // update global var to avoid cyclic imports
}

func BoolValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.BoolNull(), nil
	}
	val := types.BoolValue(from.Bool())
	if boolType, ok := to.Type(ctx).(basetypes.BoolTypable); ok {
		return boolType.ValueFromBool(ctx, val)
	}
	return val, nil
}

func IntValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.Int64Null(), nil
	}
	val := types.Int64Value(from.Int())
	if intType, ok := to.Type(ctx).(basetypes.Int64Typable); ok {
		return intType.ValueFromInt64(ctx, val)
	}
	return val, nil
}

func Uint32ValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.Int64Null(), nil
	}
	val := types.Int64Value(int64(from.Uint()))
	if intType, ok := to.Type(ctx).(basetypes.Int64Typable); ok {
		return intType.ValueFromInt64(ctx, val)
	}
	return val, nil
}

func Uint64ValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.NumberNull(), nil
	}
	val := types.NumberValue(big.NewFloat(0).SetPrec(0).SetUint64(from.Uint()))
	if numType, ok := to.Type(ctx).(basetypes.NumberTypable); ok {
		return numType.ValueFromNumber(ctx, val)
	}
	return val, nil
}

func FloatValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.Float64Null(), nil
	}
	val := types.Float64Value(from.Float())
	if floatType, ok := to.Type(ctx).(basetypes.Float64Typable); ok {
		return floatType.ValueFromFloat64(ctx, val)
	}
	return val, nil
}

func StringValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.StringNull(), nil
	}
	val := types.StringValue(from.String())
	if stringType, ok := to.Type(ctx).(basetypes.StringTypable); ok {
		return stringType.ValueFromString(ctx, val)
	}
	return val, nil
}

func BytesValueToTF(ctx context.Context, from protoreflect.Value, to attr.Value) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.StringNull(), nil
	}
	val := types.StringValue(base64.StdEncoding.EncodeToString(from.Bytes()))
	if stringType, ok := to.Type(ctx).(basetypes.StringTypable); ok {
		return stringType.ValueFromString(ctx, val)
	}
	return val, nil
}

func EnumValueToTF(
	_ context.Context,
	from protoreflect.Value,
	desc protoreflect.FieldDescriptor,
	to attr.Value,
) (attr.Value, diag.Diagnostics) {
	if !from.IsValid() {
		return types.StringNull(), nil
	}
	enumVal := desc.Enum().Values().ByNumber(from.Enum())
	var retVal types.String
	if enumVal == nil {
		retVal = types.StringValue(fmt.Sprintf("Unknown[%d]", from.Enum()))
	} else {
		retVal = types.StringValue(string(enumVal.Name()))
	}
	if stringType, ok := to.Type(context.Background()).(basetypes.StringTypable); ok {
		return stringType.ValueFromString(context.Background(), retVal)
	}
	return retVal, nil
}

func nameWithColon(name string) string {
	if name != "" {
		return name + ": "
	}
	return ""
}

func MessageValueToTFRecursive(
	ctx context.Context,
	model attr.Value,
	from proto.Message,
	name string,
	nameMap map[string]map[string]string,
) (attr.Value, diag.Diagnostics, bool) {
	typ := model.Type(ctx)
	dynType, isDynamic := typ.(basetypes.DynamicTypable)
	if wt, ok := wellknown.WellKnownOf(from.ProtoReflect().Descriptor()); ok {
		if typ.Equal(wt.Type()) {
			val, d := wt.ToValue(ctx, from)
			return val, d, !d.HasError()
		}
		if isDynamic {
			val, d := wt.ToDynamicValue(ctx, from)
			return val, d, !d.HasError()
		}
	}
	if typ.Equal(jsontypes.NormalizedType{}) {
		bytes, err := protojson.Marshal(from)
		if err != nil { // notest // hard to pass message that will fail to marshal under normal condition
			return jsontypes.NewNormalizedNull(), DiagnosticsFromErrString(
				"failed to marshal message",
				fmt.Sprintf(
					"%sfailed to marshal message to protojson %s",
					nameWithColon(name), err.Error(),
				),
			), false
		}
		return jsontypes.NewNormalizedValue(string(bytes)), diag.Diagnostics{}, true
	}
	if isDynamic {
		dynVal, d := DynamicMessageToTFRecursive(ctx, from, name, nameMap)
		val, castDiag := dynType.ValueFromDynamic(ctx, dynVal)
		d.Append(castDiag...)
		return val, d, true
	}
	objValuable, ok := model.(basetypes.ObjectValuable)
	if !ok {
		return model, DiagnosticsFromErrString(
			"wrong terraform model type",
			fmt.Sprintf(
				"%swrong terraform model type %s, expecting jsontypes."+
					"Normalized or ObjectValuable",
				nameWithColon(name), typ,
			),
		), false
	}
	objValuable, d := MessageToTFRecursive(ctx, from, objValuable, name, nameMap)
	return objValuable, d, !d.HasError()
}

func AttrTypeCheck(
	_ context.Context,
	attrType attr.Type,
	desc protoreflect.FieldDescriptor,
	name string,
) diag.Diagnostics {
	if _, isDynamic := attrType.(basetypes.DynamicTypable); isDynamic {
		return diag.Diagnostics{}
	}
	var reqType string
	ok := false
	switch desc.Kind() {
	case protoreflect.BoolKind:
		reqType = "basetypes.BoolType"
		_, ok = attrType.(basetypes.BoolTypable)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		reqType = "basetypes.Int64Type"
		_, ok = attrType.(basetypes.Int64Typable)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		reqType = "basetypes.NumberType"
		_, ok = attrType.(basetypes.NumberTypable)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		reqType = "basetypes.Float64Type"
		_, ok = attrType.(basetypes.Float64Typable)
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.EnumKind:
		reqType = "basetypes.StringType"
		_, ok = attrType.(basetypes.StringTypable)
	case protoreflect.MessageKind:
		wt, isWN := wellknown.WellKnownOf(desc.Message())
		if isWN && attrType.Equal(wt.Type()) {
			return diag.Diagnostics{}
		}
		if attrType.Equal(jsontypes.NormalizedType{}) {
			return diag.Diagnostics{}
		}
		if _, isObj := attrType.(basetypes.ObjectTypable); isObj {
			return diag.Diagnostics{}
		}
		fallthrough
	default: // notest // hard to replicate type that is not here
		return DiagnosticsFromErrString(
			"protobuf type not supported",
			fmt.Sprintf(
				"%s: protobuf type conversion to terraform %s not supported for %s",
				name, attrType, desc.Kind(),
			),
		)
	}
	if ok {
		return diag.Diagnostics{}
	}
	return DiagnosticsFromErrString(
		"protobuf type not supported",
		fmt.Sprintf(
			"%s: protobuf type conversion to terraform %s not supported for %s, required %s",
			name, attrType, desc.Kind(), reqType,
		),
	)
}

func SingleFieldToTF(
	ctx context.Context,
	to attr.Value,
	from protoreflect.Value,
	desc protoreflect.FieldDescriptor,
	name string,
	nameMap map[string]map[string]string,
) (attr.Value, diag.Diagnostics, bool) {
	ok := true
	diagnostics := AttrTypeCheck(ctx, to.Type(ctx), desc, name)
	if diagnostics.HasError() {
		return to, diagnostics, false
	}
	if from.Equal(desc.Default()) && to.IsNull() && !desc.HasPresence() {
		return to, diagnostics, true // preserve null on default values
	}
	dynType, dynamic := to.Type(ctx).(basetypes.DynamicTypable)
	switch desc.Kind() {
	case protoreflect.BoolKind:
		to, diagnostics = BoolValueToTF(ctx, from, to)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		to, diagnostics = IntValueToTF(ctx, from, to)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		to, diagnostics = Uint32ValueToTF(ctx, from, to)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		to, diagnostics = Uint64ValueToTF(ctx, from, to)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		to, diagnostics = FloatValueToTF(ctx, from, to)
	case protoreflect.StringKind:
		to, diagnostics = StringValueToTF(ctx, from, to)
	case protoreflect.BytesKind:
		to, diagnostics = BytesValueToTF(ctx, from, to)
	case protoreflect.EnumKind:
		to, diagnostics = EnumValueToTF(ctx, from, desc, to)
	case protoreflect.MessageKind:
		to, diagnostics, ok = MessageValueToTFRecursive(
			ctx, to, from.Message().Interface(), name,
			nameMap,
		)
	default: // notest // hard to mock this call
		ok = false
		diagnostics = DiagnosticsFromErrString(
			"protobuf type not supported",
			fmt.Sprintf(
				"%s: terraform to protobuf type conversion not supported for %s",
				name, desc.Kind(),
			),
		)
	}
	if dynamic && to != nil {
		var innerDiag diag.Diagnostics
		if dynTo, isDynamic := to.(basetypes.DynamicValuable); isDynamic {
			if !dynTo.Type(ctx).Equal(dynType) {
				dynValue, innerDiag := dynTo.ToDynamicValue(ctx)
				diagnostics.Append(innerDiag...)
				if innerDiag.HasError() {
					return to, diagnostics, false
				}
				to, innerDiag = dynType.ValueFromDynamic(ctx, dynValue)
				diagnostics.Append(innerDiag...)
			}
		} else {
			to, innerDiag = dynType.ValueFromDynamic(ctx, types.DynamicValue(to))
			diagnostics.Append(innerDiag...)
		}
	}
	return to, diagnostics, ok
}

func NullOfType(ctx context.Context, attrType attr.Type) (attr.Value, diag.Diagnostics) {
	if typ, ok := attrType.(basetypes.DynamicTypable); ok {
		return typ.ValueFromDynamic(ctx, types.DynamicNull())
	}
	tfVal := tftypes.NewValue(attrType.TerraformType(ctx), nil)
	val, err := attrType.ValueFromTerraform(ctx, tfVal)
	if err != nil {
		return attrType.ValueType(ctx), DiagnosticsFromErrString(
			"terraform value conversion failed",
			fmt.Sprintf(
				"failed to create nil value of type %s using terraform conversion: %s",
				attrType, err.Error(),
			),
		)
	}
	return val, diag.Diagnostics{}
}

func ListFieldToTF(
	ctx context.Context,
	val attr.Value,
	from protoreflect.Value,
	desc protoreflect.FieldDescriptor,
	name string,
	nameMap map[string]map[string]string,
) (attr.Value, diag.Diagnostics, bool) {
	typ := val.Type(ctx)
	dynType, isDynamic := typ.(basetypes.DynamicTypable)
	unwrapped, _, diagnostics := ctypes.UnwrapDynamic(ctx, val)
	listValuable, hasModel := unwrapped.(basetypes.ListValuable)
	var valListType basetypes.ListTypable
	var listModel basetypes.ListValue
	if hasModel {
		listType, ok := listValuable.Type(ctx).(basetypes.ListTypable)
		if !ok {
			return val, DiagnosticsFromErrString(
				"destination type mismatch",
				fmt.Sprintf("%s: destination value is ListValuable, but type %T is not ListTypable",
					name, listValuable.Type(ctx),
				),
			), false
		}
		listVal, innerDiag := listValuable.ToListValue(ctx)
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() {
			return val, innerDiag, false
		}
		valListType = listType
		listModel = listVal
	}
	if !hasModel && !isDynamic {
		return val, DiagnosticsFromErrString(
			"destination not a list or dynamic",
			fmt.Sprintf("%s: destination not a list or dynamic, but %s",
				name, val.Type(ctx),
			),
		), false
	}
	var elementType attr.Type = types.DynamicType
	listElements := []attr.Value{}
	if hasModel {
		if !isDynamic {
			elementType = listModel.ElementType(ctx)
		}
		listElements = listModel.Elements()
	} else if valTuple, ok := unwrapped.(basetypes.TupleValue); ok { // unlike objects, tuples can not be overridden
		listElements = valTuple.Elements()
	}
	src := from.List()
	if src.Len() == 0 && unwrapped.IsNull() {
		return val, diagnostics, true // preserve null lists as null
	}
	res := []attr.Value{}
	resTypes := []attr.Type{}
	sameType := true
	listLen := 0
	if hasModel && len(listElements) > 0 {
		listLen = len(listElements)
	}
	for i := range src.Len() {
		el := src.Get(i)
		keyName := fmt.Sprintf("[%d]", i)
		var listEl attr.Value
		if hasModel && listLen > i {
			listEl = listElements[i]
		} else {
			tfNull, innerDiag := NullOfType(ctx, elementType)
			diagnostics.Append(innerDiag...)
			listEl = tfNull
		}
		tfEl, innerDiag, ok := SingleFieldToTF(
			ctx, listEl, el, desc, name+keyName, nameMap,
		)
		diagnostics.Append(innerDiag...)
		if ok {
			unwrapped, _, innerDiag := ctypes.UnwrapDynamic(ctx, tfEl)
			diagnostics.Append(innerDiag...)
			if innerDiag.HasError() {
				continue
			}
			if _, isDyn := unwrapped.(basetypes.DynamicValuable); isDyn {
				if unwrapped.IsNull() {
					unwrapped = types.BoolNull() // arbitrary non-dynamic null, as in maps
				} else {
					diagnostics.AddError(
						"not unwrappable dynamic in list",
						fmt.Sprintf(
							"%s: list element not unwrappable: %s",
							name+keyName, tfEl,
						),
					)
					continue
				}
			}
			if sameType && len(res) > 0 && !unwrapped.Type(ctx).Equal(
				res[0].Type(ctx),
			) {
				sameType = false
			}
			resTypes = append(resTypes, unwrapped.Type(ctx))
			res = append(res, unwrapped)
		}
	}
	if !sameType {
		if !isDynamic { // technically must not be possible
			wrongTypes := make([]string, 0, len(res))
			for _, el := range res {
				if !el.Type(ctx).Equal(elementType) {
					wrongTypes = append(wrongTypes, el.Type(ctx).String())
				}
			}
			diagnostics.AddError(
				"wrong types in array",
				fmt.Sprintf("Model type is %s but found types %s",
					elementType.String(), strings.Join(wrongTypes, ", "),
				),
			)
		} else {
			tuple, innerDiag := types.TupleValue(
				resTypes, res,
			)
			diagnostics.Append(innerDiag...)
			if innerDiag.HasError() {
				return val, diagnostics, false
			}
			ret, innerDiag := dynType.ValueFromDynamic(
				ctx, types.DynamicValue(tuple),
			)
			diagnostics.Append(innerDiag...)
			if innerDiag.HasError() {
				return val, diagnostics, false
			}
			return ret, diagnostics, true
		}
	} else if len(res) > 0 {
		elementType = resTypes[0]
	}
	resList, innerDiag := types.ListValue(elementType, res)
	diagnostics.Append(innerDiag...)
	if innerDiag.HasError() {
		return val, diagnostics, false
	}
	if isDynamic {
		res, innerDiag := dynType.ValueFromDynamic(
			ctx, types.DynamicValue(resList),
		)
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() {
			return val, diagnostics, false
		}
		return res, diagnostics, true
	} else {
		res, innerDiag := valListType.ValueFromList(ctx, resList)
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() {
			return val, diagnostics, false
		}
		return res, diagnostics, true
	}
}

func MapKeyToString(from protoreflect.MapKey, name string) (string, diag.Diagnostics) {
	val := from.Interface()
	switch typedVal := val.(type) {
	case bool:
		return strconv.FormatBool(typedVal), diag.Diagnostics{}
	case int32:
		return strconv.FormatInt(int64(typedVal), 10), diag.Diagnostics{}
	case int64:
		return strconv.FormatInt(typedVal, 10), diag.Diagnostics{}
	case uint32:
		return strconv.FormatUint(uint64(typedVal), 10), diag.Diagnostics{}
	case uint64:
		return strconv.FormatUint(typedVal, 10), diag.Diagnostics{}
	case string:
		return typedVal, diag.Diagnostics{}
	default:
		return "", DiagnosticsFromErrString(
			"unsupported map key type",
			fmt.Sprintf(
				"%s: unsupported map key type %T, map key %q",
				name, typedVal, typedVal,
			),
		)
	}
}

func MapFieldToTF(
	ctx context.Context,
	val attr.Value,
	from protoreflect.Value,
	desc protoreflect.FieldDescriptor,
	name string,
	nameMap map[string]map[string]string,
) (attr.Value, diag.Diagnostics, bool) {
	typ := val.Type(ctx)
	dynType, isDynamic := typ.(basetypes.DynamicTypable)
	unwrapped, _, diagnostics := ctypes.UnwrapDynamic(ctx, val)
	if diagnostics.HasError() {
		return val, diagnostics, false
	}
	valMappable, hasModel := unwrapped.(basetypes.MapValuable)
	var valMapType basetypes.MapTypable
	var valMap basetypes.MapValue
	if hasModel {
		valMapTyp, ok := valMappable.Type(ctx).(basetypes.MapTypable)
		if hasModel && !ok {
			return val, DiagnosticsFromErrString(
				"destination type mismatch",
				fmt.Sprintf("%s: destination value is MapValuable, but type %T is not MapTypable",
					name, valMappable.Type(ctx),
				),
			), false
		}
		mapVal, innerDiag := valMappable.ToMapValue(ctx)
		diagnostics.Append(innerDiag...)
		if diagnostics.HasError() {
			return val, diagnostics, false
		}
		valMapType = valMapTyp
		valMap = mapVal
	}
	if !hasModel && !isDynamic {
		return val, DiagnosticsFromErrString(
			"unexpected destination value",
			fmt.Sprintf(
				"%s: destination value type is %s, expecting map or dynamic",
				name, typ,
			),
		), false
	}
	src := from.Map()
	if src.Len() == 0 && unwrapped.IsNull() {
		return val, diagnostics, true // preserve null maps as null
	}
	var modelType attr.Type = types.DynamicType
	elements := map[string]attr.Value{}
	if hasModel {
		elements = valMap.Elements()
		if !isDynamic {
			modelType = valMap.ElementType(ctx)
		}
	} else if valObjectable, ok := unwrapped.(basetypes.ObjectValuable); ok {
		objVal, innerDiag := valObjectable.ToObjectValue(ctx)
		diagnostics.Append(innerDiag...)
		if diagnostics.HasError() {
			return val, diagnostics, false
		}
		elements = objVal.Attributes()
	}
	result := map[string]attr.Value{}
	src.Range(func(mk protoreflect.MapKey, v protoreflect.Value) bool {
		keyName := fmt.Sprintf("[%q]", mk.Interface())
		key, innerDiag := MapKeyToString(mk, name+keyName)
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() { // notest // not mockable
			return true
		}
		priorEl, ok := elements[key]
		if !ok {
			priorEl, innerDiag = NullOfType(ctx, modelType)
			diagnostics.Append(innerDiag...)
		} else if isDynamic {
			if _, ok := priorEl.(basetypes.DynamicValuable); !ok {
				priorEl = types.DynamicValue(priorEl)
			}
		}
		keyName = fmt.Sprintf("[%q]", key)
		tfEl, innerDiag, ok := SingleFieldToTF(
			ctx, priorEl, v, desc.MapValue(), name+keyName, nameMap,
		)
		diagnostics.Append(innerDiag...)
		if ok {
			unwrapped, _, innerDiag := ctypes.UnwrapDynamic(ctx, tfEl)
			diagnostics.Append(innerDiag...)
			if innerDiag.HasError() {
				return true
			}
			if _, isDyn := unwrapped.(basetypes.DynamicValuable); isDyn {
				if unwrapped.IsNull() {
					unwrapped = types.BoolNull() // inside dynamic, any null is fine, choosing bool null
				} else {
					diagnostics.AddError(
						"dynamic value can not unwrap",
						fmt.Sprintf(
							"%s: dynamic value %T can not be unwrapped to concrete type",
							keyName, unwrapped,
						),
					)
					return true
				}
			}
			result[key] = unwrapped
		}
		return true
	})
	sameType := true
	var elementType attr.Type = nil
	elementTypes := map[string]attr.Type{}
	if hasModel && !isDynamic {
		elementType = modelType
	}
	for k, v := range result {
		typ := v.Type(ctx)
		elementTypes[k] = typ
		if elementType == nil {
			elementType = typ
			continue
		}
		if sameType && !typ.Equal(elementType) {
			sameType = false
		}
	}
	if !sameType {
		if hasModel && !isDynamic { // technically must not be possible
			wrongTypes := make([]string, 0, len(result))
			for _, v := range result {
				if !v.Type(ctx).Equal(elementType) {
					wrongTypes = append(wrongTypes, v.Type(ctx).String())
				}
			}
			diagnostics.AddError(
				"wrong types in map",
				fmt.Sprintf("Model type is %s but found types %s",
					elementType.String(), strings.Join(wrongTypes, ", "),
				),
			)
			return val, diagnostics, false
		}
		if isDynamic {
			retObj, innerDiag := types.ObjectValue(elementTypes, result)
			diagnostics.Append(innerDiag...)
			if innerDiag.HasError() {
				return val, diagnostics, false
			}
			ret, innerDiag := dynType.ValueFromDynamic(ctx, types.DynamicValue(retObj))
			diagnostics.Append(innerDiag...)
			if innerDiag.HasError() {
				return val, diagnostics, false
			}
			return ret, diagnostics, true
		}
	}
	retMap, innerDiag := types.MapValue(elementType, result)
	diagnostics.Append(innerDiag...)
	if innerDiag.HasError() {
		return val, diagnostics, false
	}
	if isDynamic {
		ret, innerDiag := dynType.ValueFromDynamic(ctx, types.DynamicValue(retMap))
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() {
			return val, diagnostics, false
		}
		return ret, diagnostics, true
	} else {
		ret, innerDiag := valMapType.ValueFromMap(ctx, retMap)
		diagnostics.Append(innerDiag...)
		if innerDiag.HasError() {
			return val, diagnostics, false
		}
		return ret, diagnostics, true
	}
}

var (
	nameToTFCache = make(map[protoreflect.FullName]string)
	nameToTFMu    sync.RWMutex
)

// getTFFieldName returns the Terraform name for a protobuf field,
// computing it once and caching it thereafter.
func getTFFieldName(
	fd protoreflect.FieldDescriptor,
	nameMap map[string]map[string]string,
) (string, error) {
	if len(nameMap) == 0 {
		return string(fd.Name()), nil
	}
	fieldNames, ok := nameMap[string(fd.Parent().FullName())]
	if !ok {
		return string(fd.Name()), nil
	}

	// Fast path: try to read under a read lock.
	nameToTFMu.RLock()
	if name, ok := nameToTFCache[fd.FullName()]; ok {
		nameToTFMu.RUnlock()
		return name, nil
	}
	nameToTFMu.RUnlock()

	tfName := string(fd.Name())
	for tfNameI, fieldName := range fieldNames {
		if fieldName == string(fd.Name()) {
			tfName = tfNameI
			break
		}
	}

	// Take the write lock only long enough to insert.
	nameToTFMu.Lock()
	nameToTFCache[fd.FullName()] = tfName
	nameToTFMu.Unlock()

	return tfName, nil
}

func DynamicMessageToTFRecursive(
	ctx context.Context,
	from protoreflect.ProtoMessage,
	path string,
	nameMap map[string]map[string]string,
) (types.Dynamic, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	if from == nil {
		return types.DynamicNull(), diagnostics
	}
	attrPathPrefix := path
	if attrPathPrefix != "" {
		attrPathPrefix = attrPathPrefix + "."
	}
	fromReflect := from.ProtoReflect()

	resultMap := map[string]attr.Value{}
	resultTypes := map[string]attr.Type{}

	fromReflect.Range(func(
		fd protoreflect.FieldDescriptor,
		v protoreflect.Value,
	) bool {
		tfName, err := getTFFieldName(fd, nameMap)
		if err != nil {
			diagnostics.AddError(
				"error getting field name",
				fmt.Sprintf(
					"Error getting field name for field %q at path %q: %s",
					fd.FullName(), path, err.Error(),
				),
			)
			return true
		}
		var val attr.Value
		var diagInner diag.Diagnostics
		var ok bool
		tmpVal := types.DynamicNull()
		if fd.IsList() {
			val, diagInner, ok = ListFieldToTF(
				ctx, tmpVal, v, fd, attrPathPrefix+tfName, nameMap,
			)
		} else if fd.IsMap() {
			val, diagInner, ok = MapFieldToTF(
				ctx, tmpVal, v, fd, attrPathPrefix+tfName, nameMap,
			)
		} else {
			val, diagInner, ok = SingleFieldToTF(
				ctx, tmpVal, v, fd, attrPathPrefix+tfName, nameMap,
			)
		}
		diagnostics.Append(diagInner...)
		if ok {
			val, _, innerDiag := ctypes.UnwrapDynamic(ctx, val)
			diagnostics.Append(innerDiag...)
			resultMap[tfName] = val
			resultTypes[tfName] = val.Type(ctx)
		}
		return true
	})
	res, diagInner := basetypes.NewObjectValue(resultTypes, resultMap)
	diagnostics.Append(diagInner...)

	return types.DynamicValue(res), diagnostics
}

func isEmptyObject(ctx context.Context, x attr.Value) (bool, bool, diag.Diagnostics) {
	obj, ok := x.(basetypes.ObjectValuable)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return false, ok, nil
	}
	objVal, d := obj.ToObjectValue(ctx)
	if d.HasError() {
		return false, ok, d
	}
	for _, x := range objVal.Attributes() {
		if !x.IsNull() && !x.IsUnknown() {
			return false, ok, nil
		}
	}
	return true, ok, nil
}

func trueEmptyObject(ctx context.Context, x attr.Value) (
	attr.Value, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	obj, ok := x.(basetypes.ObjectValuable)
	if !ok {
		diags.AddError(
			"trueEmptyObject: not an object",
			fmt.Sprintf("passed value not an object: %T", x),
		)
		return nil, diags
	}
	typ, ok := obj.Type(ctx).(basetypes.ObjectTypable)
	if !ok {
		diags.AddError(
			"trueEmptyObject: not an object type",
			fmt.Sprintf("passed value type not an object type: %T", obj.Type(ctx)),
		)
		return nil, diags
	}
	objVal, d := obj.ToObjectValue(ctx)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	attrs := objVal.AttributeTypes(ctx)
	retAttrs := map[string]attr.Value{}
	for name, attr := range attrs {
		val, innerDiag := NullOfType(ctx, attr)
		diags.Append(innerDiag...)
		retAttrs[name] = val
	}
	retObj, innerDiag := types.ObjectValue(attrs, retAttrs)
	diags.Append(innerDiag...)
	ret, innerDiag := typ.ValueFromObject(ctx, retObj)
	diags.Append(innerDiag...)
	return ret, diags
}

func GetAnnotation[T any](
	options proto.Message,
	xt protoreflect.ExtensionType,
) (_ T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(
				"panic while getting extension %s from %s: %s",
				xt.TypeDescriptor().FullName(),
				options.ProtoReflect().Descriptor().FullName(),
				r,
			)
		}
	}()
	retInterface := proto.GetExtension(options, xt)
	ret, ok := retInterface.(T)
	if !ok {
		return ret, fmt.Errorf("expected %T, got %T", ret, retInterface)
	}
	return ret, nil
}

func IsMeaningfulEmpty(desc protoreflect.FieldDescriptor) (
	bool, diag.Diagnostics,
) {
	anno, err := GetAnnotation[[]nebius.FieldBehavior](
		desc.Options(), nebius.E_FieldBehavior,
	)
	if err != nil {
		ret := diag.Diagnostics{}
		ret.AddError("Annotation fetch error",
			fmt.Sprintf("Error: %s", err.Error()))
		return true, ret
	}
	for _, a := range anno {
		if a == nebius.FieldBehavior_MEANINGFUL_EMPTY_VALUE {
			return true, nil
		}
	}
	return false, nil
}

func IsInputOnly(desc protoreflect.FieldDescriptor) (
	bool, diag.Diagnostics,
) {
	anno, err := GetAnnotation[[]nebius.FieldBehavior](
		desc.Options(), nebius.E_FieldBehavior,
	)
	if err != nil {
		ret := diag.Diagnostics{}
		ret.AddError("Annotation fetch error",
			fmt.Sprintf("Error: %s", err.Error()))
		return true, ret
	}
	for _, a := range anno {
		if a == nebius.FieldBehavior_INPUT_ONLY {
			return true, nil
		}
	}
	return false, nil
}

func IsSensitive(desc protoreflect.FieldDescriptor) (
	bool, diag.Diagnostics,
) {
	anno, err := GetAnnotation[bool](desc.Options(), nebius.E_Sensitive)
	if err != nil {
		ret := diag.Diagnostics{}
		ret.AddError("Annotation fetch error",
			fmt.Sprintf("Error: %s", err.Error()))
		return true, ret
	}
	return anno, nil
}

func MessageToTFRecursive(
	ctx context.Context,
	from protoreflect.ProtoMessage,
	to basetypes.ObjectValuable,
	path string,
	nameMap map[string]map[string]string,
) (basetypes.ObjectValuable, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	if from == nil {
		return to, diagnostics
	}
	typ, ok := to.Type(ctx).(basetypes.ObjectTypable)
	if !ok {
		diagnostics.AddError(
			"wrong terraform model type",
			fmt.Sprintf(
				"%swrong terraform model type %s, expecting types.ObjectTypable",
				nameWithColon(path), to.Type(ctx),
			),
		)
		return to, diagnostics
	}
	toObj, convDiag := to.ToObjectValue(ctx)
	diagnostics.Append(convDiag...)
	if convDiag.HasError() {
		return to, diagnostics
	}
	attributes := toObj.Attributes()
	attributeTypes := toObj.AttributeTypes(ctx)
	attrPathPrefix := path
	if attrPathPrefix != "" {
		attrPathPrefix = attrPathPrefix + "."
	}
	fromReflect := from.ProtoReflect()
	fromDesc := fromReflect.Descriptor()

	resultMap := map[string]attr.Value{}

	for key, attrType := range attributeTypes {
		attr, ok := attributes[key]
		if !ok {
			var innerDiag diag.Diagnostics
			attr, innerDiag = NullOfType(ctx, attrType)
			diagnostics.Append(innerDiag...)
		}
		pbName, err := fieldNameByTFNameAndParent(key, fromDesc, nameMap)
		if err != nil {
			diagnostics.AddError(
				"get field name error",
				fmt.Sprintf("get field name for %q in %s: %s", key,
					fromDesc.FullName(), err.Error(),
				),
			)
			continue
		}
		resultMap[key] = attr
		protoFieldDesc := fromDesc.Fields().ByName(pbName)
		if protoFieldDesc == nil {
			continue
		}
		inputOnly, innerDiag := IsInputOnly(protoFieldDesc)
		diagnostics.Append(innerDiag...)
		if inputOnly && attr.IsNull() {
			sensitive, innerDiag := IsSensitive(protoFieldDesc)
			diagnostics.Append(innerDiag...)
			if sensitive {
				// the field probably set using a write-only parameter
				continue
			}
		}
		if !fromReflect.Has(protoFieldDesc) && protoFieldDesc.HasPresence() {
			if !attr.IsNull() && !attr.IsUnknown() {
				// Skip setting input-only field if the returned value is null
				if inputOnly {
					continue
				}
			}
			isME, innerDiag := IsMeaningfulEmpty(protoFieldDesc)
			diagnostics.Append(innerDiag...)
			if !isME && !attr.IsNull() && !attr.IsUnknown() {
				isEmpty, isObj, innerDiag := isEmptyObject(ctx, attr)
				diagnostics.Append(innerDiag...)
				if isObj && isEmpty {
					// if it's not meaningful empty value, we return the same
					// type of empty as we received:
					// If we received null, we return null
					// If we received empty object, we return empty object
					// The only issue, in the source we could have had unknowns
					// That's why we construct a true empty object here
					trueEmpty, innerDiag := trueEmptyObject(ctx, attr)
					diagnostics.Append(innerDiag...)
					resultMap[key] = trueEmpty
					continue
				}
				if !isObj && protoFieldDesc.Kind() == protoreflect.MessageKind {
					wt, ok := wellknown.WellKnownOf(protoFieldDesc.Message())
					if ok {
						// preserve semantically equal values
						semEq, inner := SemanticallyEqual(ctx, attr, wt.Empty())
						diagnostics.Append(inner...)
						if semEq {
							continue
						}
					}
				}
			}
			attr, innerDiag := NullOfType(ctx, attrType)
			diagnostics.Append(innerDiag...)
			resultMap[key] = attr
			continue
		}
		protoField := fromReflect.Get(protoFieldDesc)
		if protoFieldDesc.IsList() {
			val, diagInner, ok := ListFieldToTF(
				ctx, attr, protoField, protoFieldDesc, attrPathPrefix+key,
				nameMap,
			)
			diagnostics.Append(diagInner...)
			if ok {
				if inputOnly {
					if vlist, ok := val.(types.List); ok &&
						!attr.IsUnknown() &&
						!vlist.IsNull() && !vlist.IsUnknown() &&
						len(vlist.Elements()) == 0 {
						continue // empty list on input only — skip setting
					}
				}
				resultMap[key] = val
			}
			continue
		}
		if protoFieldDesc.IsMap() {
			val, diagInner, ok := MapFieldToTF(
				ctx, attr, protoField, protoFieldDesc, attrPathPrefix+key,
				nameMap,
			)
			diagnostics.Append(diagInner...)
			if ok {
				if inputOnly {
					if vmap, ok := val.(types.Map); ok &&
						!attr.IsUnknown() &&
						!vmap.IsNull() && !vmap.IsUnknown() &&
						len(vmap.Elements()) == 0 {
						continue // empty map on input only — skip setting
					}
				}
				resultMap[key] = val
			}
			continue
		}
		// Skip setting input-only field if the returned value is default
		if inputOnly && !attr.IsUnknown() &&
			protoField.Equal(protoFieldDesc.Default()) &&
			!protoFieldDesc.HasPresence() {
			continue
		}
		val, diagInner, ok := SingleFieldToTF(
			ctx, attr, protoField, protoFieldDesc, attrPathPrefix+key,
			nameMap,
		)
		diagnostics.Append(diagInner...)
		if ok {
			if attr.IsNull() {
				isME, innerDiag := IsMeaningfulEmpty(protoFieldDesc)
				diagnostics.Append(innerDiag...)
				if !isME {
					isEmpty, isObj, innerDiag := isEmptyObject(ctx, val)
					diagnostics.Append(innerDiag...)
					if isObj && isEmpty {
						resultMap[key] = attr
						continue
					}
					if !isObj && protoFieldDesc.Kind() == protoreflect.MessageKind {
						wt, ok := wellknown.WellKnownOf(protoFieldDesc.Message())
						if ok {
							// preserve null if received semantically empty value
							semEq, inner := SemanticallyEqual(ctx, val, wt.Empty())
							diagnostics.Append(inner...)
							if semEq {
								resultMap[key] = attr
								continue
							}
						}
					}
				}
			}
			resultMap[key] = val
		}
	}
	resObj, diagInner := basetypes.NewObjectValue(attributeTypes, resultMap)
	diagnostics.Append(diagInner...)
	res, convDiag := typ.ValueFromObject(ctx, resObj)
	diagnostics.Append(convDiag...)
	return res, diagnostics
}

func MessageValueToTF(
	ctx context.Context,
	model attr.Value,
	from proto.Message,
	nameMap map[string]map[string]string,
) (attr.Value, diag.Diagnostics) {
	ret, diag, _ := MessageValueToTFRecursive(ctx, model, from, "", nameMap)
	return ret, diag
}

func DynamicMessageToTF(
	ctx context.Context,
	from protoreflect.ProtoMessage,
	nameMap map[string]map[string]string,
) (types.Dynamic, diag.Diagnostics) {
	return DynamicMessageToTFRecursive(ctx, from, "", nameMap)
}

func MessageToTF(
	ctx context.Context,
	from protoreflect.ProtoMessage,
	to basetypes.ObjectValuable,
	nameMap map[string]map[string]string,
) (basetypes.ObjectValuable, diag.Diagnostics) {
	return MessageToTFRecursive(ctx, from, to, "", nameMap)
}
