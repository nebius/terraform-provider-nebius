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

func (r *commonResource) addWriteOnlyFields(
	ctx context.Context,
	config tfsdk.Config,
	data *types.Object,
) (*types.Object, *mask.Mask, diag.Diagnostics) {
	if !r.provider.WriteOnlyFieldsSupported() {
		return data, nil, diag.Diagnostics{}
	}
	var diags diag.Diagnostics

	woMask, err := r.implementation.WriteOnlyFields()
	if err != nil {
		diags = ErrorToDiag(diags, err, "failed to get write only fields mask")
		return data, nil, diags
	}
	if woMask == nil { // no write only fields, nothing to do
		return data, nil, diags
	}

	var dataMirror types.Object
	innerDiag := config.GetAttribute(
		ctx, path.Root(writeonly.FieldName), &dataMirror,
	)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return data, nil, diags
	}

	data, unk, dataUnk, innerDiag := writeonly.ParseWriteOnlyFields(
		ctx, data, dataMirror,
		woMask, mask.NewFieldPath(), path.Empty(),
	)
	diags.Append(innerDiag...)
	if diags.HasError() {
		return data, dataUnk, diags
	}
	unk = ctypes.AppendUnknownMask(unk, mask.FieldPath{}, dataUnk)
	return data, unk, diags
}
