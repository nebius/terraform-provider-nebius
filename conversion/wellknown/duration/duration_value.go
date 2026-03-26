package duration

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	_ basetypes.StringValuableWithSemanticEquals = (*Duration)(nil)
	_ xattr.ValidateableAttribute                = (*Duration)(nil)
)

// Duration represents a valid Duration string. Semantic equality
// logic is defined for Duration such that inconsequential differences are
// ignored.
type Duration struct {
	basetypes.StringValue
}

// ValidateAttribute implements xattr.ValidateableAttribute.
func (v *Duration) ValidateAttribute(
	ctx context.Context,
	req xattr.ValidateAttributeRequest,
	resp *xattr.ValidateAttributeResponse,
) {
	if v.IsUnknown() || v.IsNull() {
		return
	}

	if _, err := ParseDuration(v.ValueString()); err != nil {
		resp.Diagnostics.Append(diag.WithPath(req.Path, diag.NewErrorDiagnostic(
			"Invalid Duration String Value",
			"A string value was provided that is not valid Duration string"+
				" format.\n\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)))

		return
	}
}

// Type returns an DurationType.
func (v Duration) Type(_ context.Context) attr.Type {
	return DurationType{}
}

// Equal returns true if the given value is equivalent.
func (v Duration) Equal(o attr.Value) bool {
	other, ok := o.(Duration)

	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

func (v Duration) StringSemanticEquals(
	_ context.Context,
	newValuable basetypes.StringValuable,
) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(Duration)
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

	// Duration strings are already validated at this point, ignoring errors
	newDuration, _ := ParseDuration(newValue.ValueString())
	currentDuration, _ := ParseDuration(v.ValueString())

	if newDuration.GetSeconds() != currentDuration.GetSeconds() {
		return false, diags
	}
	if newDuration.GetNanos() != currentDuration.GetNanos() {
		return false, diags
	}

	return true, diags
}

// ValueDuration creates a new *durationpb.Duration instance with the Duration
// StringValue. A null or unknown value will produce an error diagnostic.
func (v Duration) ValueDuration() (*durationpb.Duration, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() {
		diags.Append(diag.NewErrorDiagnostic("Duration ValueDuration Error",
			"Duration string value is null",
		))
		return nil, diags
	}

	if v.IsUnknown() {
		diags.Append(diag.NewErrorDiagnostic("Duration ValueDuration Error",
			"Duration string value is unknown",
		))
		return nil, diags
	}

	d, err := ParseDuration(v.ValueString())
	if err != nil {
		diags.Append(diag.NewErrorDiagnostic("Duration ValueDuration Error",
			err.Error(),
		))
		return nil, diags
	}

	return d, nil
}

func NewDurationNull() Duration {
	return Duration{
		StringValue: basetypes.NewStringNull(),
	}
}

func NewDurationUnknown() Duration {
	return Duration{
		StringValue: basetypes.NewStringUnknown(),
	}
}

func NewDurationEmpty() Duration {
	return Duration{
		StringValue: basetypes.NewStringValue("0s"),
	}
}

// NewDurationStringValue creates a Duration with a known value or raises an error
// diagnostic if the string is not Duration format.
func NewDurationStringValue(value string) (Duration, diag.Diagnostics) {
	_, err := ParseDuration(value)

	if err != nil {
		// Returning an unknown value will guarantee that, as a last resort,
		// Terraform will return an error if attempting to store into state.
		return NewDurationUnknown(), diag.Diagnostics{diag.NewErrorDiagnostic(
			"Invalid Duration String Value",
			"A string value was provided that is not valid Duration string"+
				" format.\n\n"+
				"Given Value: "+value+"\n"+
				"Error: "+err.Error(),
		)}
	}

	return Duration{
		StringValue: basetypes.NewStringValue(value),
	}, nil
}

// NewDurationStringValueMust creates a Duration with a known value or raises a panic
// if the string is not ISO-8601 format.
//
// This creation function is only recommended to create Duration values which
// either will not potentially affect practitioners, such as testing, or within
// exhaustively tested provider logic.
func NewDurationStringValueMust(value string) Duration {
	_, err := ParseDuration(value)

	if err != nil {
		panic(fmt.Sprintf("Invalid Duration String (%s): %s", value, err))
	}

	return Duration{
		StringValue: basetypes.NewStringValue(value),
	}
}

// NewDurationValue creates a Duration with a known value.
func NewDurationValue(value *durationpb.Duration) Duration {
	if value == nil {
		return NewDurationNull()
	}
	return Duration{
		StringValue: basetypes.NewStringValue(FormatDuration(value)),
	}
}
