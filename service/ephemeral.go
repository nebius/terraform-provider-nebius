package service

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/proto"

	"github.com/nebius/terraform-provider-nebius/conversion"
	"github.com/nebius/terraform-provider-nebius/service/requestcontext"
)

type SimpleEphemeralResourceInterface interface {
	GetName() string
	EphemeralResourceSchema() schema.Schema
	ReadRequestMessage() proto.Message
	Read(context.Context, proto.Message) (
		proto.Message,
		*requestcontext.Context,
		error,
	)
	FieldNameMap() map[string]map[string]string
}

type simpleEphemeralResource struct {
	implementation SimpleEphemeralResourceInterface
}

func (s *simpleEphemeralResource) Metadata(
	_ context.Context,
	req ephemeral.MetadataRequest,
	resp *ephemeral.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_" + s.implementation.GetName()
}

func (s *simpleEphemeralResource) Open(
	ctx context.Context,
	req ephemeral.OpenRequest,
	resp *ephemeral.OpenResponse,
) {
	reqCtx := s.readRequest(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			s.implementation.EphemeralResourceSchema().Type(),
			s.implementation.FieldNameMap(),
		)
	}
}

func (s *simpleEphemeralResource) readRequest(
	ctx context.Context,
	req ephemeral.OpenRequest,
	resp *ephemeral.OpenResponse,
) *requestcontext.Context {
	var data types.Object

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	readReq := s.implementation.ReadRequestMessage()
	_, diag := conversion.MessageFromTF(
		ctx, data, readReq, s.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	readResp, reqCtx, err := s.implementation.Read(ctx, readReq)
	if err != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "ephemeral resource read failed",
		)
		return reqCtx
	}

	tmpObjValue, diag := conversion.MessageToTF(
		ctx, readResp, data, s.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return reqCtx
	}
	tmpObj, diag := tmpObjValue.ToObjectValue(ctx)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return reqCtx
	}
	data = tmpObj

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	return reqCtx

}

func (s *simpleEphemeralResource) Schema(
	_ context.Context,
	_ ephemeral.SchemaRequest,
	resp *ephemeral.SchemaResponse,
) {
	resp.Schema = s.implementation.EphemeralResourceSchema()
}

func NewSimpleEphemeralResource(implementation SimpleEphemeralResourceInterface) ephemeral.EphemeralResource {
	return &simpleEphemeralResource{
		implementation: implementation,
	}
}
