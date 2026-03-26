package service

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/proto"

	"github.com/nebius/terraform-provider-nebius/conversion"
	"github.com/nebius/terraform-provider-nebius/service/requestcontext"
)

type SimpleDataSourceInterface interface {
	GetName() string
	DataSourceSchema() schema.Schema
	ReadRequestMessage() proto.Message
	Read(context.Context, proto.Message) (
		proto.Message,
		*requestcontext.Context,
		error,
	)
	FieldNameMap() map[string]map[string]string
}

type simpleDataSource struct {
	implementation SimpleDataSourceInterface
}

// Metadata implements datasource.DataSource.
func (s *simpleDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_" + s.implementation.GetName()
}

// Read implements datasource.DataSource.
func (s *simpleDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	reqCtx := s.readRequest(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			s.implementation.DataSourceSchema().Type(),
			s.implementation.FieldNameMap(),
		)
	}
}

func (s *simpleDataSource) readRequest(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
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
			resp.Diagnostics, err, "datasource read failed",
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	return reqCtx

}

// Schema implements datasource.DataSource.
func (s *simpleDataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = s.implementation.DataSourceSchema()
}

func NewSimpleDataSource(implementation SimpleDataSourceInterface) datasource.DataSource {
	return &simpleDataSource{
		implementation: implementation,
	}
}
