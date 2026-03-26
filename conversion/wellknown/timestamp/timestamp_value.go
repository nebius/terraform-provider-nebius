package timestamp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	_ basetypes.StringValuableWithSemanticEquals = (*TimeStamp)(nil)
	_ xattr.ValidateableAttribute                = (*TimeStamp)(nil)
)

var beginningOfEpoch = (&timestamppb.Timestamp{}).AsTime()

// TimeStamp represents a valid ISO-8601-formatted string. Semantic equality
// logic is defined for ISO-8601 such that inconsequential differences between
// the `Z` suffix and a `00:00` UTC offset are ignored.
type TimeStamp struct {
	basetypes.StringValue
}

// ValidateAttribute implements [xattr.ValidateableAttribute]. This type requires the value provided to
// be a String value that is valid ISO-8601 format. This utilizes the Go `time`
// library which does not strictly adhere to the ISO-8601 standard and may allow
// strings that are not valid ISO-8601 strings
func (v TimeStamp) ValidateAttribute(
	ctx context.Context,
	req xattr.ValidateAttributeRequest,
	resp *xattr.ValidateAttributeResponse,
) {

	if v.IsUnknown() || v.IsNull() {
		return
	}

	valueString := v.ValueString()
	if valueString == "" {
		return
	}

	if _, err := time.Parse(time.RFC3339Nano, valueString); err != nil {
		resp.Diagnostics.Append(diag.WithPath(req.Path, diag.NewErrorDiagnostic(
			"Invalid TimeStamp String Value",
			"A string value was provided that is not valid ISO-8601 string"+
				" format.\n\n"+
				"Given Value: "+valueString+"\n"+
				"Error: "+err.Error(),
		)))

		return
	}
}

// Type returns an RFC3339Type.
func (v TimeStamp) Type(_ context.Context) attr.Type {
	return TimeStampType{}
}

// Equal returns true if the given value is equivalent.
func (v TimeStamp) Equal(o attr.Value) bool {
	other, ok := o.(TimeStamp)

	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals returns true if the given ISO-8601 string value is
// semantically equal to the current ISO-8601 string value. This comparison
// utilizes time.Parse to create time.Time instances and then compares the
// resulting RFC 3339-formatted string representations. This means ISO-8601
// values utilizing the `Z` Zulu suffix as an offset are considered semantically
// equal to ISO-8601 that define a `00:00` UTC offset.
//
// Examples:
//   - `2023-07-25T20:43:16+00:00` is semantically equal to
//     `2023-07-25T20:43:16Z`
//   - `2023-07-25T20:43:16-00:00` is semantically equal to
//     `2023-07-25T20:43:16Z` - while ISO-8601 defines an unknown local offset
//     (`-00:00`) to be different from an offset of `Z`, time.Parse converts
//     `-00:00` to `+00:00` during parsing.
//
// Counterexamples:
//   - `2023-07-25T23:43:16+00:00` expresses the same time as
//     `2023-07-25T20:43:16-03:00` but is NOT considered to be semantically
//     equal.
func (v TimeStamp) StringSemanticEquals(
	_ context.Context,
	newValuable basetypes.StringValuable,
) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(TimeStamp)
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

	// RFC3339 strings are already validated at this point, ignoring errors
	newTime := beginningOfEpoch
	currentTime := beginningOfEpoch
	if newValue.ValueString() != "" {
		newTime, _ = time.Parse(time.RFC3339Nano, newValue.ValueString())
	}
	if v.ValueString() != "" {
		currentTime, _ = time.Parse(time.RFC3339Nano, v.ValueString())
	}

	return currentTime.Format(time.RFC3339Nano) == newTime.Format(time.RFC3339Nano), diags
}

// ValueTime creates a new time.Time instance with the TimeStamp StringValue.
// A null or unknown value will produce an error diagnostic.
func (v TimeStamp) ValueTime() (time.Time, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() {
		diags.Append(diag.NewErrorDiagnostic("TimeStamp ValueTime Error",
			"TimeStamp string value is null",
		))
		return beginningOfEpoch, diags
	}

	if v.IsUnknown() {
		diags.Append(diag.NewErrorDiagnostic("TimeStamp ValueTime Error",
			"TimeStamp string value is unknown",
		))
		return beginningOfEpoch, diags
	}

	if v.ValueString() == "" {
		return beginningOfEpoch, nil
	}

	t, err := time.Parse(time.RFC3339Nano, v.ValueString())
	if err != nil {
		diags.Append(diag.NewErrorDiagnostic("TimeStamp ValueTime Error",
			err.Error(),
		))
		return beginningOfEpoch, diags
	}

	return t, nil
}

// NewTimeStampNull creates a TimeStamp with a null value. Determine whether
// the value is null via IsNull method.
func NewTimeStampNull() TimeStamp {
	return TimeStamp{
		StringValue: basetypes.NewStringNull(),
	}
}

// NewTimeStampEmpty creates a TimeStamp with an empty value.
func NewTimeStampEmpty() TimeStamp {
	return TimeStamp{
		StringValue: basetypes.NewStringValue("1970-01-01T00:00:00Z"),
	}
}

// NewTimeStampUnknown creates a TimeStamp with an unknown value. Determine
// whether the value is unknown via IsUnknown method.
func NewTimeStampUnknown() TimeStamp {
	return TimeStamp{
		StringValue: basetypes.NewStringUnknown(),
	}
}

// NewTimeStampTimeValue creates a TimeStamp with a known value.
func NewTimeStampTimeValue(value time.Time) TimeStamp {
	return TimeStamp{
		StringValue: basetypes.NewStringValue(value.Format(time.RFC3339Nano)),
	}
}

// NewTimeStampTimePointerValue creates a TimeStamp with a null value if nil or
// a known value.
func NewTimeStampTimePointerValue(value *time.Time) TimeStamp {
	if value == nil {
		return NewTimeStampNull()
	}

	return TimeStamp{
		StringValue: basetypes.NewStringValue(value.Format(time.RFC3339Nano)),
	}
}

// NewTimeStampValue creates a TimeStamp with a known value or raises an error
// diagnostic if the string is not ISO-8601 format.
func NewTimeStampValue(value string) (TimeStamp, diag.Diagnostics) {
	if value == "" {
		return TimeStamp{
			StringValue: basetypes.NewStringValue(value),
		}, nil
	}
	_, err := time.Parse(time.RFC3339Nano, value)

	if err != nil {
		// Returning an unknown value will guarantee that, as a last resort,
		// Terraform will return an error if attempting to store into state.
		return NewTimeStampUnknown(), diag.Diagnostics{diag.NewErrorDiagnostic(
			"Invalid TimeStamp String Value",
			"A string value was provided that is not valid ISO-8601 string"+
				" format.\n\n"+
				"Given Value: "+value+"\n"+
				"Error: "+err.Error(),
		)}
	}

	return TimeStamp{
		StringValue: basetypes.NewStringValue(value),
	}, nil
}

// NewTimeStampValueMust creates a TimeStamp with a known value or raises a panic
// if the string is not ISO-8601 format.
//
// This creation function is only recommended to create TimeStamp values which
// either will not potentially affect practitioners, such as testing, or within
// exhaustively tested provider logic.
func NewTimeStampValueMust(value string) TimeStamp {
	if value == "" {
		return TimeStamp{
			StringValue: basetypes.NewStringValue(value),
		}
	}
	_, err := time.Parse(time.RFC3339Nano, value)

	if err != nil {
		panic(fmt.Sprintf("Invalid TimeStamp String (%s): %s", value, err))
	}

	return TimeStamp{
		StringValue: basetypes.NewStringValue(value),
	}
}

// NewTimeStampPointerValue creates a TimeStamp with a null value if nil, a known
// value, or raises an error diagnostic if the string is not ISO-8601 format.
func NewTimeStampPointerValue(value *string) (TimeStamp, diag.Diagnostics) {
	if value == nil {
		return NewTimeStampNull(), nil
	}

	return NewTimeStampValue(*value)
}

// NewTimeStampPointerValueMust creates an TimeStamp with a null value if nil, a
// known value, or raises a panic if the string is not ISO-8601 format.
//
// This creation function is only recommended to create TimeStamp values which
// either will not potentially affect practitioners, such as testing, or within
// exhaustively tested provider logic.
func NewTimeStampPointerValueMust(value *string) TimeStamp {
	if value == nil {
		return NewTimeStampNull()
	}

	return NewTimeStampValueMust(*value)
}
