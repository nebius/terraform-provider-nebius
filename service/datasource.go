package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/proto"

	"github.com/nebius/gosdk/constants"
	common "github.com/nebius/gosdk/proto/nebius/common/v1"
	"github.com/nebius/terraform-provider-nebius/conversion"
	"github.com/nebius/terraform-provider-nebius/provider"
	"github.com/nebius/terraform-provider-nebius/service/requestcontext"
)

type AdditionalGetterFunc func(
	ctx context.Context,
	input proto.Message,
) (
	*common.ResourceMetadata,
	proto.Message,
	proto.Message,
	*requestcontext.Context,
	error,
)
type AdditionalGetter struct {
	Name         string
	InputMessage func() proto.Message
	GetterFunc   AdditionalGetterFunc
}

type DataSourceInterface interface {
	GetName() string
	DataSourceSchema() schema.Schema
	Read(ctx context.Context, id string) (
		*common.ResourceMetadata,
		proto.Message,
		proto.Message,
		*requestcontext.Context,
		error,
	)
	GetAdditionalGetters() map[string]AdditionalGetter
	FieldNameMap() map[string]map[string]string
}

type GetByNameInterface interface {
	GetByName(ctx context.Context, name, parentID string) (
		*common.ResourceMetadata,
		proto.Message,
		proto.Message,
		*requestcontext.Context,
		error,
	)
}

type commonDataSource struct {
	implementation DataSourceInterface
}

func NewDataSource(
	implementation DataSourceInterface,
	_ provider.Provider,
) datasource.DataSource {
	return &commonDataSource{
		implementation: implementation,
	}
}

// Metadata implements datasource.DataSource.
func (r *commonDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_" + r.implementation.GetName()
}

// Schema implements datasource.DataSource.
func (r *commonDataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = r.implementation.DataSourceSchema()
}

func (r *commonDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	reqCtx := r.getAndSave(ctx, req, resp)
	if reqCtx != nil {
		resp.Diagnostics = reqCtx.WrapDiagnostics(
			resp.Diagnostics,
			r.implementation.DataSourceSchema().Type(),
			r.implementation.FieldNameMap(),
		)
	}
}

func (r *commonDataSource) getFromServer(ctx context.Context, data types.Object) (
	*common.ResourceMetadata,
	proto.Message,
	proto.Message,
	*requestcontext.Context,
	diag.Diagnostics,
) {
	diags := diag.Diagnostics{}

	pid, diag := getPStringFromObject(ctx, data, constants.FieldID, path.Empty())
	diags.Append(diag...)
	if diag.HasError() {
		return nil, nil, nil, nil, diags
	}
	if pid != nil {
		otherFields := whatIsSetBut(data, constants.FieldID)
		if len(otherFields) > 0 {
			diags.AddAttributeError(
				path.Root(constants.FieldID),
				"too many fields set",
				"either unset `id` or `"+strings.Join(otherFields, "`, `")+"`",
			)
			return nil, nil, nil, nil, diags
		}
		if *pid == "" {
			diags.AddAttributeError(
				path.Root(constants.FieldID),
				"id is empty", "id must not be empty",
			)
			return nil, nil, nil, nil, diags
		}
		metadata, spec, status, reqCtx, err := r.implementation.Read(ctx, *pid)
		return metadata, spec, status, reqCtx, ErrorToDiagPath(
			diags, err, "datasource reading by ID failed",
			path.Root(constants.FieldID),
		)
	}

	additionalGetters := r.implementation.GetAdditionalGetters()

	getByNameable, hasGBN := r.implementation.(GetByNameInterface)

	needToBeSet := "the `id`"

	if hasGBN {
		needToBeSet += ", or both `parent_id` and `name`"

		pname, diag := getPStringFromObject(ctx, data, constants.FieldName, path.Empty())
		diags.Append(diag...)
		if diag.HasError() {
			return nil, nil, nil, nil, diags
		}
		ppid, diag := getPStringFromObject(ctx, data, constants.FieldParentID, path.Empty())
		diags.Append(diag...)
		if diag.HasError() {
			return nil, nil, nil, nil, diags
		}

		if ppid != nil && pname != nil {
			otherFields := whatIsSetBut(
				data, constants.FieldName, constants.FieldParentID,
			)
			if len(otherFields) > 0 {
				diags.AddError(
					"too many fields set",
					"either unset `parent_id` and `name`, or `"+
						strings.Join(otherFields, "`, `")+"`",
				)
				return nil, nil, nil, nil, diags
			}

			if *ppid == "" {
				diags.AddAttributeError(
					path.Root(constants.FieldParentID),
					"empty parent_id",
					"set both `parent_id` and `name` to get the data source by"+
						" name, or set other fields instead",
				)
				return nil, nil, nil, nil, diags
			}
			if *pname == "" {
				diags.AddAttributeError(
					path.Root(constants.FieldName),
					"empty name",
					"set both `parent_id` and `name` to get the data source by"+
						" name, or set other fields instead",
				)
				return nil, nil, nil, nil, diags
			}
			metadata, spec, status, reqCtx, err := getByNameable.GetByName(
				ctx, *pname, *ppid,
			)
			return metadata, spec, status, reqCtx, ErrorToDiag(
				diags, err, "datasource reading by name failed",
			)
		}
	}

	if len(additionalGetters) > 0 {
		getterNames := make([]string, 0, len(additionalGetters))
		for getterName, getter := range additionalGetters {
			getterNames = append(getterNames, getterName)
			objVal, ok := data.Attributes()[getterName]
			objPath := path.Root(getterName)
			if !ok {
				continue
			}
			obj, ok := objVal.(types.Object)
			if !ok {
				diags.AddAttributeError(
					objPath,
					"invalid getter",
					fmt.Sprintf("getter %q is not an object", getterName),
				)
				return nil, nil, nil, nil, diags
			}
			if !isKnown(obj) {
				continue
			}
			otherFields := whatIsSetBut(data, getterName)
			if len(otherFields) > 0 {
				diags.AddAttributeError(
					objPath,
					"too many fields set",
					"either unset `"+getterName+"`, or `"+
						strings.Join(otherFields, "`, `")+"`",
				)
				return nil, nil, nil, nil, diags
			}
			input := getter.InputMessage()
			_, innerDiag := conversion.MessageFromTFPath(
				ctx, obj, input, objPath, r.implementation.FieldNameMap(),
			)
			diags.Append(innerDiag...)
			if innerDiag.HasError() {
				return nil, nil, nil, nil, diags
			}
			metadata, spec, status, reqCtx, err := getter.GetterFunc(ctx, input)
			return metadata, spec, status, reqCtx, ErrorToDiag(
				diags, err,
				"datasource reading using getter "+getterName+" failed",
			)
		}
		if len(additionalGetters) > 1 {
			needToBeSet += ", or one of `" + strings.Join(getterNames, "`, `") +
				"` getters"
		} else {
			needToBeSet += ", or the `" + getterNames[0] + "` getter"
		}
	}

	diags.AddError(
		"no fields set",
		needToBeSet+" must be set to get the data source",
	)
	return nil, nil, nil, nil, diags
}

// Read implements datasource.DataSource.
func (r *commonDataSource) getAndSave(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) *requestcontext.Context {
	var data types.Object

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return nil
	}
	metadata, spec, status, reqCtx, diag := r.getFromServer(ctx, data)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
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
