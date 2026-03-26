package anytf

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
)

const (
	anyTypeAttribute  = "type_url"
	anyValueAttribute = "value"
)

var (
	_ basetypes.ObjectValuableWithSemanticEquals = (*Any)(nil)
	_ xattr.ValidateableAttribute                = (*Any)(nil)
)

var DynamicMessageToTF func(
	context.Context,
	proto.Message,
	map[string]map[string]string,
) (types.Dynamic, diag.Diagnostics)
var DynamicMessageFromTF func(
	ctx context.Context,
	from basetypes.DynamicValuable,
	to proto.Message,
	nameMap map[string]map[string]string,
) (*mask.Mask, diag.Diagnostics)

// Any represents a valid Any object. Semantic equality
// logic is defined for Any such that inconsequential differences are
// ignored.
type Any struct {
	basetypes.ObjectValue
}

// ValidateAttribute implements xattr.ValidateableAttribute.
func (v *Any) ValidateAttribute(
	ctx context.Context,
	req xattr.ValidateAttributeRequest,
	resp *xattr.ValidateAttributeResponse,
) {
	if v.IsUnknown() || v.IsNull() {
		return
	}

	var obj struct {
		TypeURL types.String  `tfsdk:"type_url"`
		Value   types.Dynamic `tfsdk:"value"`
	}

	if diag := v.As(ctx, &obj, basetypes.ObjectAsOptions{}); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		resp.Diagnostics.AddAttributeError(req.Path,
			"AnyType Type Validation Error",
			fmt.Sprintf(
				"Parsing as %T failed", obj,
			),
		)
		return
	}
	unwrapped, _, innerDiag := ctypes.UnwrapDynamic(ctx, obj.Value)
	resp.Diagnostics.Append(innerDiag...)
	if unwrapped.IsNull() || unwrapped.IsUnknown() ||
		obj.TypeURL.IsNull() || obj.TypeURL.IsUnknown() {
		return
	}
	switch val := unwrapped.(type) {
	case types.String:
		_, err := base64.StdEncoding.DecodeString(val.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Any ValueMessage Error",
				fmt.Sprintf("Value string to bytes base64 decode error: %s", err),
			)
			return
		}
	case types.Object:
		_, _, innerDiag := convertDynamic(
			ctx, nil, obj.TypeURL.ValueString(), val,
		)
		resp.Diagnostics.Append(innerDiag...)
		return
	default:
		resp.Diagnostics.AddError(
			"Any ValueMessage Error",
			fmt.Sprintf("Unsupported value type: %s", val.Type(ctx).String()),
		)
		return
	}
}

type Resolver interface {
	FindMessageByURL(url string) (protoreflect.MessageType, error)
}

var anyAttrTypes = map[string]attr.Type{
	anyTypeAttribute:  types.StringType,
	anyValueAttribute: types.DynamicType,
}

// Type returns an AnyType.
func (v Any) Type(_ context.Context) attr.Type {
	return AnyTypeType
}

// Equal returns true if the given value is equivalent.
func (v Any) Equal(o attr.Value) bool {
	other, ok := o.(Any)

	if !ok {
		return false
	}

	return v.ObjectValue.Equal(other.ObjectValue)
}

func (v Any) ObjectSemanticEquals(
	ctx context.Context,
	o basetypes.ObjectValuable,
) (bool, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	other, ok := o.(Any)

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
	vAnypb, vUnk, innerDiag := v.ValueAnypb(ctx, nil)
	diags.Append(innerDiag...)
	oAnypb, oUnk, innerDiag := other.ValueAnypb(ctx, nil)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return false, diags
	}

	if vUnk != nil || oUnk != nil {
		return true, diags
	}

	return proto.Equal(vAnypb, oAnypb), diags
}

func (v Any) fetchValue(ctx context.Context) (
	string, attr.Value, *mask.Mask, diag.Diagnostics,
) {
	var diags diag.Diagnostics

	if v.IsNull() {
		diags.Append(diag.NewErrorDiagnostic("Any ValueMessage Error",
			"Any value is null",
		))
		return "", nil, nil, diags
	}

	if v.IsUnknown() {
		diags.Append(diag.NewErrorDiagnostic("Any ValueMessage Error",
			"Any value is unknown",
		))
		return "", nil, nil, diags
	}

	typeURL, ok := v.ObjectValue.Attributes()[anyTypeAttribute]
	if !ok {
		diags.Append(diag.NewErrorDiagnostic("Any ValueMessage Error",
			"Any type_url is not present",
		))
		return "", nil, nil, diags
	}
	if typeURL.IsUnknown() {
		var unk *mask.Mask
		unk = ctypes.AppendUnknownPath(unk, mask.FieldPath{mask.FieldKey(anyTypeAttribute)})
		return "", nil, unk, diags
	}
	typeURLStrVal, ok := typeURL.(types.String)
	if !ok {
		diags.Append(diag.NewErrorDiagnostic("Any ValueMessage Error",
			"Any type_url is not a string type",
		))
		return "", nil, nil, diags
	}
	typeURLStr := typeURLStrVal.ValueString()
	valueAttr, ok := v.ObjectValue.Attributes()[anyValueAttribute]
	if !ok {
		diags.Append(diag.NewErrorDiagnostic("Any ValueMessage Error",
			"Any value is missing",
		))
		return typeURLStr, nil, nil, diags
	}
	unwrapped, _, innerDiag := ctypes.UnwrapDynamic(ctx, valueAttr)
	diags.Append(innerDiag...)
	if unwrapped.IsUnknown() {
		var unk *mask.Mask
		unk = ctypes.AppendUnknownPath(unk, mask.FieldPath{mask.FieldKey(anyValueAttribute)})
		return typeURLStr, nil, unk, diags
	}
	return typeURLStr, unwrapped, nil, diags
}

func createMessage(
	resolver Resolver,
	typeURL string,
) (proto.Message, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if resolver == nil {
		resolver = protoregistry.GlobalTypes
	}
	mt, err := resolver.FindMessageByURL(typeURL)
	if err != nil {
		diags.Append(diag.NewErrorDiagnostic(
			"Any ValueMessage Error",
			fmt.Sprintf(
				"Couldn't resolve %q: %s", typeURL, err.Error(),
			),
		))
		return nil, diags
	}
	dst := mt.New().Interface()
	return dst, diags
}

func convertDynamic(
	ctx context.Context,
	resolver Resolver,
	typeURL string,
	val types.Object,
) (proto.Message, *mask.Mask, diag.Diagnostics) {
	dst, diags := createMessage(resolver, typeURL)
	if diags.HasError() {
		return nil, nil, diags
	}
	unk, innerDiag := DynamicMessageFromTF(ctx, types.DynamicValue(val), dst, nil)
	diags.Append(innerDiag...)
	return dst, unk, diags
}

// ValueMessage creates a new proto.Message instance from this Any.
// If a resolver is nil, than it will use the default resolver
// [protoregistry.GlobalTypes].
// A null or unknown value or null/unknown attributes will produce an error
// diagnostic, except null value that will create (proto.Message)(nil).
func (v Any) ValueMessage(ctx context.Context, resolver Resolver) (
	proto.Message, *mask.Mask, diag.Diagnostics,
) {
	typeURLStr, unwrapped, unk, diags := v.fetchValue(ctx)
	if diags.HasError() {
		return nil, nil, diags
	}
	if unk != nil {
		return nil, unk, diags
	}
	dst, innerDiag := createMessage(resolver, typeURLStr)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return nil, nil, diags
	}
	if unwrapped.IsNull() {
		valueType := reflect.TypeOf(dst).Elem()
		nullValue := reflect.New(valueType).Interface().(proto.Message)
		return nullValue, nil, diags
	}
	switch val := unwrapped.(type) {
	case types.String:
		bytes, err := base64.StdEncoding.DecodeString(val.ValueString())
		if err != nil {
			diags.AddError(
				"Any ValueMessage Error",
				fmt.Sprintf("Value string to bytes base64 decode error: %s", err),
			)
			return nil, nil, diags
		}
		err = proto.Unmarshal(bytes, dst)
		if err != nil {
			diags.AddError(
				"Any ValueMessage Error",
				fmt.Sprintf("Unmarshaling error: %s", err),
			)
			return nil, nil, diags
		}
		return dst, nil, diags
	case types.Object:
		ret, unk, innerDiag := convertDynamic(ctx, resolver, typeURLStr, val)
		diags.Append(innerDiag...)
		var retUnk *mask.Mask
		retUnk = ctypes.AppendUnknownMask(
			retUnk,
			mask.FieldPath{mask.FieldKey("value")},
			unk,
		)
		return ret, retUnk, diags
	default:
		diags.AddError(
			"Any ValueMessage Error",
			fmt.Sprintf("Unsupported value type: %T", val),
		)
		return nil, nil, diags
	}
}

func (v Any) ValueAnypb(ctx context.Context, resolver Resolver) (
	*anypb.Any, *mask.Mask, diag.Diagnostics,
) {
	typeURLStr, unwrapped, unk, diags := v.fetchValue(ctx)
	if diags.HasError() {
		return nil, nil, diags
	}
	if unk != nil {
		return nil, unk, diags
	}
	if unwrapped.IsNull() {
		diags.AddError(
			"Any ValueMessage Error",
			"Value is null",
		)
		return nil, nil, diags
	}
	switch val := unwrapped.(type) {
	case types.String:
		bytes, err := base64.StdEncoding.DecodeString(val.ValueString())
		if err != nil {
			diags.AddError(
				"Any ValueMessage Error",
				fmt.Sprintf("Value string to bytes base64 decode error: %s", err),
			)
			return nil, nil, diags
		}
		return &anypb.Any{
			TypeUrl: typeURLStr,
			Value:   bytes,
		}, nil, diags
	case types.Object:
		msg, unk, innerDiag := convertDynamic(ctx, resolver, typeURLStr, val)
		diags.Append(innerDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		ret, err := anypb.New(msg)
		if err != nil {
			diags.AddError("Any ValueAnypb Error", fmt.Sprintf(
				"Failed to create anypb.Any from message: %s", err.Error(),
			))
		}
		var retUnk *mask.Mask
		retUnk = ctypes.AppendUnknownMask(
			retUnk,
			mask.FieldPath{mask.FieldKey("value")},
			unk,
		)
		return ret, retUnk, diags
	default:
		diags.AddError(
			"Any ValueMessage Error",
			fmt.Sprintf("Unsupported value type: %T", val),
		)
		return nil, nil, diags
	}
}

func NewAnyNull() Any {
	return Any{
		ObjectValue: basetypes.NewObjectNull(anyAttrTypes),
	}
}

func NewAnyUnknown() Any {
	return Any{
		ObjectValue: basetypes.NewObjectUnknown(anyAttrTypes),
	}
}

func NewAnyEmpty() Any {
	return Any{
		ObjectValue: basetypes.NewObjectValueMust(
			anyAttrTypes, map[string]attr.Value{
				anyTypeAttribute:  types.StringValue(""),
				anyValueAttribute: types.DynamicValue(types.StringValue("")),
			},
		),
	}
}

func NewAnypbValue(ctx context.Context, value *anypb.Any) (
	Any, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if value == nil {
		return NewAnyNull(), diags
	}
	msg, err := value.UnmarshalNew()
	if err != nil {
		ret, innerDiag := basetypes.NewObjectValue(anyAttrTypes,
			map[string]attr.Value{
				anyTypeAttribute: types.StringValue(value.GetTypeUrl()),
				anyValueAttribute: types.DynamicValue(types.StringValue(
					base64.StdEncoding.EncodeToString(value.GetValue()),
				)),
			},
		)
		diags.Append(innerDiag...)
		return Any{
			ObjectValue: ret,
		}, innerDiag
	}
	return NewMessageValue(ctx, msg)
}

// NewMessageValue creates a Any from a message.
func NewMessageValue(ctx context.Context, value proto.Message) (
	Any, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if value == nil {
		return NewAnyNull(), diags
	}
	const urlPrefix = "type.googleapis.com/"
	typeURL := urlPrefix + string(value.ProtoReflect().Descriptor().FullName())
	valueDyn, innerDiag := DynamicMessageToTF(ctx, value, nil)
	diags.Append(innerDiag...)
	ret, innerDiag := basetypes.NewObjectValue(anyAttrTypes,
		map[string]attr.Value{
			anyTypeAttribute:  types.StringValue(typeURL),
			anyValueAttribute: valueDyn,
		},
	)
	diags.Append(innerDiag...)

	return Any{
		ObjectValue: ret,
	}, diags
}
