package wellknown

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/anytf"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/duration"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/structtf"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/timestamp"
	"github.com/nebius/terraform-provider-nebius/tftype"
)

type WellKnown interface {
	TFType() tftype.TFType
	Documentation() string
	Type() attr.Type
	FromValue(context.Context, attr.Value) (
		proto.Message, *mask.Mask, diag.Diagnostics,
	)
	ToValue(context.Context, proto.Message) (attr.Value, diag.Diagnostics)
	ToDynamicValue(context.Context, proto.Message) (
		basetypes.DynamicValue, diag.Diagnostics,
	)
	Null() attr.Value
	Unknown() attr.Value
	Message() proto.Message
	Empty() attr.Value
}

type Attributable interface {
	AttributeTypes() map[string]attr.Type
}

var wellKnownTypes = []WellKnown{
	&timestamp.TimeStampType{},
	&duration.DurationType{},
	&anytf.AnyTypeType,
	&structtf.StructTypeType,
	&structtf.ValueTypeType,
}

var wellKnownMap map[protoreflect.FullName]WellKnown

func init() {
	wellKnownMap = make(map[protoreflect.FullName]WellKnown)
	for _, t := range wellKnownTypes {
		wellKnownMap[t.Message().ProtoReflect().Descriptor().FullName()] = t
	}
}

func WellKnownByName(name protoreflect.FullName) WellKnown {
	return wellKnownMap[name]
}

func WellKnownOf(msg protoreflect.MessageDescriptor) (WellKnown, bool) {
	t, ok := wellKnownMap[msg.FullName()]
	return t, ok
}
