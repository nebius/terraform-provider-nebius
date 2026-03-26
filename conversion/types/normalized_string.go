package types

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/nebius/terraform-provider-nebius/conversion/normalizers"
)

var (
	_ basetypes.StringValuableWithSemanticEquals = (*NormalizedString)(nil)
	_ xattr.ValidateableAttribute                = (*NormalizedString)(nil)
)

type NormalizedString struct {
	basetypes.StringValue
	normalizers []normalizers.Normalizer
}

func (v NormalizedString) Type(_ context.Context) attr.Type {
	return NormalizedStringType{
		normalizers: v.normalizers,
	}
}
func (v NormalizedString) Equal(o attr.Value) bool {
	other, ok := o.(NormalizedString)

	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v NormalizedString) Normalized() (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	normalizedValue := v.ValueString()

	for _, normalizer := range v.normalizers {
		if normalizer == nil {
			continue
		}
		var normDiags diag.Diagnostics
		normalizedValue, normDiags = normalizer.Normalize(normalizedValue)
		diags.Append(normDiags...)
		if diags.HasError() {
			return "", diags
		}
	}

	return normalizedValue, diags
}

func (v NormalizedString) ValidateAttribute(
	ctx context.Context,
	req xattr.ValidateAttributeRequest,
	resp *xattr.ValidateAttributeResponse,
) {
	if v.IsUnknown() || v.IsNull() {
		return
	}
	_, diags := v.Normalized()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (v NormalizedString) StringSemanticEquals(
	_ context.Context,
	newValuable basetypes.StringValuable,
) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(NormalizedString)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic "+
				"equality checks. Please report this to the provider "+
				"developers.\n\nExpected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)

		return false, diags
	}
	currentNormalized, currentDiags := v.Normalized()
	diags.Append(currentDiags...)
	if diags.HasError() {
		return false, diags
	}
	newNormalized, newDiags := newValue.Normalized()
	diags.Append(newDiags...)
	if diags.HasError() {
		return false, diags
	}

	return currentNormalized == newNormalized, diags
}

var (
	_ basetypes.StringTypable = (*NormalizedStringType)(nil)
)

type NormalizedStringType struct {
	basetypes.StringType
	normalizers []normalizers.Normalizer
}

func (t NormalizedStringType) ValueType(_ context.Context) attr.Value {
	return &NormalizedString{
		normalizers: t.normalizers,
	}
}

func (t NormalizedStringType) Documentation() string {
	docs := make([]string, 0, len(t.normalizers))
	for _, normalizer := range t.normalizers {
		if normalizer != nil {
			docs = append(docs, normalizer.Documentation())
		}
	}
	return "A normalized string with custom normalization rules applied in order: " + strings.Join(docs, " ")
}

func (t NormalizedStringType) String() string {
	names := make([]string, len(t.normalizers))
	for i, normalizer := range t.normalizers {
		if normalizer == nil {
			names[i] = "<nil>"
		} else {
			names[i] = normalizer.Name()
		}
	}
	return "types.NormalizedStringType(" + strings.Join(names, ", ") + ")"
}

func (t NormalizedStringType) Equal(o attr.Type) bool {
	other, ok := o.(NormalizedStringType)

	if !ok {
		return false
	}

	if len(t.normalizers) != len(other.normalizers) {
		return false
	}

	for i, normalizer := range t.normalizers {
		if normalizer == nil || other.normalizers[i] == nil {
			return normalizer == other.normalizers[i]
		}
		if normalizer.Name() != other.normalizers[i].Name() {
			return false
		}
	}

	return t.StringType.Equal(other.StringType)
}

func (t NormalizedStringType) ValueFromString(
	_ context.Context,
	in basetypes.StringValue,
) (basetypes.StringValuable, diag.Diagnostics) {
	return NormalizedString{
		StringValue: in,
		normalizers: t.normalizers,
	}, nil
}

func (t NormalizedStringType) FromString(in string) basetypes.StringValuable {
	return NormalizedString{
		StringValue: basetypes.NewStringValue(in),
		normalizers: t.normalizers,
	}
}

func (t NormalizedStringType) FromStringPointer(in *string) basetypes.StringValuable {
	return NormalizedString{
		StringValue: basetypes.NewStringPointerValue(in),
		normalizers: t.normalizers,
	}
}

func (t NormalizedStringType) NullValue() basetypes.StringValuable {
	return NormalizedString{
		StringValue: basetypes.NewStringNull(),
		normalizers: t.normalizers,
	}
}

func (t NormalizedStringType) UnknownValue() basetypes.StringValuable {
	return NormalizedString{
		StringValue: basetypes.NewStringUnknown(),
		normalizers: t.normalizers,
	}
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to
// convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t NormalizedStringType) ValueFromTerraform(
	ctx context.Context,
	in tftypes.Value,
) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)

	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)

	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)

	if diags.HasError() {
		return nil, fmt.Errorf(
			"converting StringValue to StringValuable: %v",
			diags,
		)
	}

	return stringValuable, nil
}

func NormalizedFromRegistry(names ...string) (NormalizedStringType, error) {
	normalizersList := make([]normalizers.Normalizer, 0, len(names))
	for _, name := range names {
		normalizer := normalizers.Get(name)
		if normalizer == nil {
			return NormalizedStringType{}, fmt.Errorf("unknown normalizer name: %s", name)
		}
		normalizersList = append(normalizersList, normalizer)
	}
	return NormalizedStringType{
		normalizers: normalizersList,
	}, nil
}

func NormalizedFromRegistryMust(names ...string) NormalizedStringType {
	normType, err := NormalizedFromRegistry(names...)
	if err != nil {
		panic(err)
	}
	return normType
}
