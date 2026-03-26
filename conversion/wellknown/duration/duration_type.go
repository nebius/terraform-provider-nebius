package duration

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/tftype"
)

var (
	_ basetypes.StringTypable = (*DurationType)(nil)
)

type DurationType struct {
	basetypes.StringType
}

func (t DurationType) TFType() tftype.TFType {
	return tftype.TFString
}

func (t DurationType) Documentation() string {
	return "Duration as a string: " +
		"possibly signed sequence of decimal numbers, each with optional " +
		"fraction and a unit suffix, such as `300ms`, `-1.5h` or `2h45m`. " +
		"Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`, `d`."
}

func (t DurationType) String() string {
	return "duration.DurationType"
}

func (t DurationType) Equal(o attr.Type) bool {
	other, ok := o.(DurationType)

	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t DurationType) Type() attr.Type {
	return t
}

func (t DurationType) FromValue(
	ctx context.Context, val attr.Value,
) (proto.Message, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	ts, ok := val.(Duration)
	if !ok {
		unwrapped, _, unwrapDiag := types.UnwrapDynamic(ctx, val)
		diags.Append(unwrapDiag...)
		if diags.HasError() {
			return nil, nil, diags
		}
		if unwrapped.IsNull() || unwrapped.IsUnknown() {
			return (*durationpb.Duration)(nil), nil, diags
		}
		str, ok := unwrapped.(basetypes.StringValuable)
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
		stringValue, innerDiag := str.ToStringValue(ctx)
		diags = append(diags, innerDiag...)
		if innerDiag.HasError() {
			return nil, nil, diags
		}
		ts = Duration{
			StringValue: stringValue,
		}
	}
	if ts.IsNull() || ts.IsUnknown() {
		return (*durationpb.Duration)(nil), nil, diag.Diagnostics{}
	}
	d, innerDiag := ts.ValueDuration()
	diags = append(diags, innerDiag...)
	if innerDiag.HasError() {
		return nil, nil, diags
	}
	return d, nil, diags
}
func (t DurationType) ToValue(_ context.Context, msg proto.Message) (
	attr.Value, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return t.Null(), diags
	}
	d, ok := msg.(*durationpb.Duration)
	if !ok {
		diags.AddError(
			"message is not *durationpb.Duration",
			fmt.Sprintf(
				"message has to be *durationpb.Duration, %T found",
				msg,
			),
		)
		return nil, diags
	}
	if d == nil {
		return t.Null(), diags
	}
	return NewDurationValue(d), diags
}
func (t DurationType) ToDynamicValue(
	ctx context.Context,
	msg proto.Message,
) (basetypes.DynamicValue, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return basetypes.NewDynamicNull(), diags
	}
	d, innerDiag := t.ToValue(ctx, msg)
	diags = append(diags, innerDiag...)
	if innerDiag.HasError() {
		return basetypes.NewDynamicNull(), diags
	}
	str, diags := d.(Duration).StringValue, diags
	if diags.HasError() {
		return basetypes.NewDynamicNull(), diags
	}
	return basetypes.NewDynamicValue(str), diags
}

func (t DurationType) Null() attr.Value {
	return NewDurationNull()
}
func (t DurationType) Unknown() attr.Value {
	return NewDurationUnknown()
}
func (t DurationType) Message() proto.Message {
	return &durationpb.Duration{}
}

func (t DurationType) Empty() attr.Value {
	return NewDurationEmpty()
}

// ValueFromString returns a StringValuable type given a StringValue.
func (t DurationType) ValueFromString(
	ctx context.Context,
	in basetypes.StringValue,
) (basetypes.StringValuable, diag.Diagnostics) {
	return Duration{
		StringValue: in,
	}, nil
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to
// convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t DurationType) ValueFromTerraform(
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
			"unexpected error converting StringValue to StringValuable: %v",
			diags,
		)
	}

	return stringValuable, nil
}
