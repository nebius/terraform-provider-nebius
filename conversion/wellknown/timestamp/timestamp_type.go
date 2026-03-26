package timestamp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/tftype"
)

var (
	_ basetypes.StringTypable = (*TimeStampType)(nil)
)

type TimeStampType struct {
	basetypes.StringType
}

func (t TimeStampType) TFType() tftype.TFType {
	return tftype.TFString
}

// ValueType returns the Value type.
func (t TimeStampType) ValueType(ctx context.Context) attr.Value {
	return TimeStamp{}
}

func (t TimeStampType) Documentation() string {
	return "A string representing a timestamp in " +
		"[ISO 8601](https://en.wikipedia.org/wiki/ISO_8601) format: " +
		"`YYYY-MM-DDTHH:MM:SSZ` or `YYYY-MM-DDTHH:MM:SS.SSS±HH:MM`"
}

func (t TimeStampType) String() string {
	return "timestamp.TimeStampType"
}

func (t TimeStampType) Equal(o attr.Type) bool {
	other, ok := o.(TimeStampType)

	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t TimeStampType) Type() attr.Type {
	return t
}

func (t TimeStampType) FromValue(
	ctx context.Context, val attr.Value,
) (proto.Message, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	ts, ok := val.(TimeStamp)
	if !ok {
		unwrapped, _, unwrapDiag := types.UnwrapDynamic(ctx, val)
		diags = append(diags, unwrapDiag...)
		if unwrapDiag.HasError() {
			return nil, nil, diags
		}
		if unwrapped.IsNull() || unwrapped.IsUnknown() {
			return (*timestamppb.Timestamp)(nil), nil, diags
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
		ts = TimeStamp{
			StringValue: stringValue,
		}
	}
	if ts.IsNull() || ts.IsUnknown() {
		return (*timestamppb.Timestamp)(nil), nil, diag.Diagnostics{}
	}
	time, innerDiag := ts.ValueTime()
	diags = append(diags, innerDiag...)
	if innerDiag.HasError() {
		return nil, nil, diags
	}
	return timestamppb.New(time), nil, diags
}
func (t TimeStampType) ToValue(_ context.Context, msg proto.Message) (
	attr.Value, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	if msg == nil {
		return t.Null(), diags
	}
	ts, ok := msg.(*timestamppb.Timestamp)
	if !ok {
		diags.AddError(
			"message is not *timestamppb.Timestamp",
			fmt.Sprintf(
				"message has to be *timestamppb.Timestamp, %T found",
				msg,
			),
		)
		return nil, diags
	}
	if ts == nil {
		return t.Null(), diags
	}
	return NewTimeStampTimeValue(ts.AsTime()), diags
}
func (t TimeStampType) ToDynamicValue(
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
	ret, innerDiag := d.(TimeStamp).ToStringValue(ctx)
	diags.Append(innerDiag...)
	return basetypes.NewDynamicValue(ret), diags
}

func (t TimeStampType) Null() attr.Value {
	return NewTimeStampNull()
}
func (t TimeStampType) Unknown() attr.Value {
	return NewTimeStampUnknown()
}
func (t TimeStampType) Message() proto.Message {
	return &timestamppb.Timestamp{}
}
func (t TimeStampType) Empty() attr.Value {
	return NewTimeStampEmpty()
}

// ValueFromString returns a StringValuable type given a StringValue.
func (t TimeStampType) ValueFromString(
	ctx context.Context,
	in basetypes.StringValue,
) (basetypes.StringValuable, diag.Diagnostics) {
	return TimeStamp{
		StringValue: in,
	}, nil
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to
// convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t TimeStampType) ValueFromTerraform(
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
