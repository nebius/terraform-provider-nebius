package service

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/proto"

	"github.com/nebius/gosdk/constants"
	"github.com/nebius/gosdk/proto/fieldmask/mask"
	common "github.com/nebius/gosdk/proto/nebius/common/v1"
	"github.com/nebius/terraform-provider-nebius/conversion"
	"github.com/nebius/terraform-provider-nebius/provider"
	"github.com/nebius/terraform-provider-nebius/service/requestcontext"
)

const (
	ReadOnCreateTimeout = 5 * time.Second
)

type ResourceInterface interface {
	GetName() string
	ResourceSchema() schema.Schema
	SpecMessage() proto.Message
	Create(
		ctx context.Context,
		metadata *common.ResourceMetadata,
		spec proto.Message,
		wellKnownID string,
	) (string, *requestcontext.Context, error)
	Read(ctx context.Context, id string) (
		*common.ResourceMetadata, proto.Message, proto.Message,
		*requestcontext.Context, error,
	)
	Update(
		ctx context.Context,
		metadata *common.ResourceMetadata,
		spec proto.Message,
	) (*requestcontext.Context, error)
	Delete(ctx context.Context, id string) (
		*requestcontext.Context, error,
	)
	WriteOnlyFields() (*mask.Mask, error)
	FieldNameMap() map[string]map[string]string
}

type commonResource struct {
	implementation ResourceInterface
	provider       provider.Provider
}

func NewResource(
	implementation ResourceInterface,
	provider provider.Provider,
) resource.Resource {
	return &commonResource{
		implementation: implementation,
		provider:       provider,
	}
}

// Metadata implements resource.Resource.
func (r *commonResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_" + r.implementation.GetName()
}

// Schema implements resource.Resource.
func (r *commonResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = r.implementation.ResourceSchema()
}

func (r *commonResource) createRequest(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) *requestcontext.Context {

	var data types.Object

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	dataPtr, _, innerDiag := r.addWriteOnlyFields(
		ctx, req.Config, &data,
	)
	resp.Diagnostics.Append(innerDiag...)
	if resp.Diagnostics.HasError() {
		return nil
	}
	if dataPtr == nil {
		resp.Diagnostics.AddError(
			"no data pointer",
			"in create, addWriteOnlyFields returned a nil pointer when non-nil data was passed",
		)
		return nil
	}
	dataWithWriteOnly := *dataPtr

	metadata, diag := metadataFromTF(
		ctx, dataWithWriteOnly, r.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	spec := r.implementation.SpecMessage()
	_, diag = conversion.MessageFromTF(
		ctx, dataWithWriteOnly, spec, r.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	wellKnownID := ""
	wkiAttr, ok := dataWithWriteOnly.Attributes()[constants.FieldWellKnownID]
	if ok && !wkiAttr.IsNull() {
		if wkiAttr.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(constants.FieldWellKnownID),
				constants.FieldWellKnownID+" is unknown",
				constants.FieldWellKnownID+" must be known on create",
			)
			return nil
		}
		wkiTFString, ok := wkiAttr.(types.String)
		if !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root(constants.FieldWellKnownID),
				constants.FieldWellKnownID+" is not a string",
				constants.FieldWellKnownID+" must be a string",
			)
			return nil
		}
		wellKnownID = wkiTFString.ValueString()
	}

	id, createReqCtx, err := r.implementation.Create(
		ctx, metadata, spec, wellKnownID,
	)
	if id != "" {
		resp.State.SetAttribute(ctx, path.Root(constants.FieldID), id)
		ctxGet, cancel := DelayedContext(
			ctx, // original context may be cancelled
			ReadOnCreateTimeout,
		)
		defer cancel()
		ctx = ctxGet
	}
	if err != nil {
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource creation failed",
		)
		if id == "" {
			return createReqCtx
		}
	}
	metadata, spec, status, readReqCtx, err := r.implementation.Read(ctx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
		}
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource read failed",
		)
		return requestcontext.MergeContexts(createReqCtx, readReqCtx)
	}
	data, diag = convertToObject(
		ctx, metadata, spec, status, data, r.implementation.FieldNameMap(),
	) // no write-only fields here, for security reasons
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return requestcontext.MergeContexts(createReqCtx, readReqCtx)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	return requestcontext.MergeContexts(createReqCtx, readReqCtx)
}

// Create implements resource.Resource.
func (r *commonResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	reqCtx := r.createRequest(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			r.implementation.ResourceSchema().Type(),
			r.implementation.FieldNameMap(),
		)
	}
}

func (r *commonResource) readRequest(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) *requestcontext.Context {
	var data types.Object

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	id, diag := getIDFromObject(ctx, data, path.Empty())
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	metadata, spec, status, reqCtx, err := r.implementation.Read(ctx, id)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return reqCtx
		}

		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource reading failed",
		)
		return reqCtx
	}
	data, diag = convertToObject(
		ctx, metadata, spec, status, data, r.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return reqCtx
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	return reqCtx
}

// Read implements resource.Resource.
func (r *commonResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) {
	reqCtx := r.readRequest(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			r.implementation.ResourceSchema().Type(),
			r.implementation.FieldNameMap(),
		)
	}
}

func (r *commonResource) updateRequest(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) *requestcontext.Context {
	var data types.Object

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	dataPtr, _, innerDiag := r.addWriteOnlyFields(
		ctx, req.Config, &data,
	)
	resp.Diagnostics.Append(innerDiag...)
	if resp.Diagnostics.HasError() {
		return nil
	}
	if dataPtr == nil {
		resp.Diagnostics.AddError(
			"no data pointer",
			"in update, addWriteOnlyFields returned a nil pointer when non-nil data was passed",
		)
		return nil
	}
	dataWithWriteOnly := *dataPtr

	metadata, diag := metadataFromTF(
		ctx, dataWithWriteOnly, r.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	spec := r.implementation.SpecMessage()
	_, diag = conversion.MessageFromTF(
		ctx, dataWithWriteOnly, spec, r.implementation.FieldNameMap(),
	)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	reqCtx, err := r.implementation.Update(ctx, metadata, spec)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
		}
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource update failed",
		)
		return reqCtx
	}
	metadata, spec, status, reqCtx, err := r.implementation.Read(
		ctx, metadata.GetId(),
	)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
		}
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource read failed",
		)
		return reqCtx
	}
	data, diag = convertToObject(
		ctx, metadata, spec, status, data, r.implementation.FieldNameMap(),
	) // no write-only fields here, for security reasons
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return reqCtx
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	return reqCtx
}

// Update implements resource.Resource.
func (r *commonResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	reqCtx := r.updateRequest(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			r.implementation.ResourceSchema().Type(),
			r.implementation.FieldNameMap(),
		)
	}
}

func (r *commonResource) deleteRequest(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) *requestcontext.Context {
	var data types.Object

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	id, diag := getIDFromObject(ctx, data, path.Empty())
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return nil
	}

	reqCtx, err := r.implementation.Delete(ctx, id)
	if err != nil {
		if isNotFoundError(err) {
			return reqCtx
		}
		resp.Diagnostics = ErrorToDiag(
			resp.Diagnostics, err, "resource deletion failed",
		)
	}
	return reqCtx
}

// Delete implements resource.Resource.
func (r *commonResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	reqCtx := r.deleteRequest(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			r.implementation.ResourceSchema().Type(),
			r.implementation.FieldNameMap(),
		)
	}
}

func (r *commonResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root(constants.FieldID), req, resp)
}
