package service

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/conversion/writeonly"
)

type writeOnlyData struct {
	value              *types.Object
	planUnknowns       *mask.Mask
	configuredUnknowns *mask.Mask
	supplied           *mask.Mask
}

func (r *commonResource) addWriteOnlyFields(
	ctx context.Context,
	config tfsdk.Config,
	data *types.Object,
) (writeOnlyData, diag.Diagnostics) {
	if !r.provider.WriteOnlyFieldsSupported() {
		return writeOnlyData{value: data}, nil
	}
	var diags diag.Diagnostics

	woMask, err := r.implementation.WriteOnlyFields()
	if err != nil {
		diags = ErrorToDiag(diags, err, "failed to get write only fields mask")
		return writeOnlyData{value: data}, diags
	}
	if woMask == nil { // no write only fields, nothing to do
		return writeOnlyData{value: data}, diags
	}

	var dataMirror types.Object
	innerDiag := config.GetAttribute(
		ctx, path.Root(writeonly.FieldName), &dataMirror,
	)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return writeOnlyData{value: data}, diags
	}

	data, unk, dataUnk, supplied, innerDiag := writeonly.ParseWriteOnlyFields(
		ctx, data, dataMirror,
		woMask, mask.NewFieldPath(), path.Empty(),
	)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return writeOnlyData{
			value:              data,
			planUnknowns:       dataUnk,
			configuredUnknowns: unk,
			supplied:           supplied,
		}, diags
	}
	allUnknowns := ctypes.AppendUnknownMask(nil, mask.FieldPath{}, unk)
	allUnknowns = ctypes.AppendUnknownMask(allUnknowns, mask.FieldPath{}, dataUnk)
	return writeOnlyData{
		value:              data,
		planUnknowns:       allUnknowns,
		configuredUnknowns: unk,
		supplied:           supplied,
	}, diags
}
