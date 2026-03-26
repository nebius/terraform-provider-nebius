package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/nebius/gosdk/constants"
	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/gosdk/proto/fieldmask/protobuf"
	common "github.com/nebius/gosdk/proto/nebius/common/v1"
	"github.com/nebius/gosdk/serviceerror"
	"github.com/nebius/terraform-provider-nebius/conversion"
)

func isNotFoundError(err error) bool {
	return status.Code(err) == codes.NotFound
}

func isKnown(a attr.Value) bool {
	return !a.IsNull() && !a.IsUnknown()
}

func whatIsSetBut(data types.Object, attrs ...string) []string {
	attrsSet := map[string]struct{}{}
	for _, attr := range attrs {
		attrsSet[attr] = struct{}{}
	}
	ret := []string{}
	for attrName, attr := range data.Attributes() {
		if _, ok := attrsSet[attrName]; ok {
			continue
		}
		if !attr.IsNull() && !attr.IsUnknown() {
			ret = append(ret, attrName)
		}
	}
	sort.Strings(ret)
	return ret
}

func getPStringFromObject(ctx context.Context, data types.Object, name string, root path.Path) (
	*string, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	obj, ok := data.Attributes()[name]
	if !ok {
		diags.AddAttributeError(
			root.AtName(name),
			name+" not found",
			fmt.Sprintf("no %s in data object", name))
		return nil, diags
	}
	valStr, ok := obj.(types.String)
	if !ok {
		diags.AddAttributeError(
			root.AtName(name),
			name+" not of type string",
			fmt.Sprintf(
				"%s must be of string type, found: %s",
				name, obj.Type(ctx).String(),
			),
		)
		return nil, diags
	}
	if !isKnown(valStr) {
		return nil, diags
	}
	return valStr.ValueStringPointer(), diags
}

func ErrorToDiag(
	diags diag.Diagnostics,
	err error,
	summary string,
) diag.Diagnostics {
	if err == nil {
		return diags
	}
	var serr *serviceerror.Error
	if errors.As(err, &serr) {
		diags.AddError(
			summary,
			fmt.Sprintf(
				"%s: %s", summary, err.Error(),
			),
		)
	} else {
		diags.AddError(
			summary,
			summary+": "+err.Error(),
		)
	}
	return diags
}

func ErrorToDiagPath(
	diags diag.Diagnostics,
	err error,
	summary string,
	attrPath path.Path,
) diag.Diagnostics {
	if err == nil {
		return diags
	}
	diags.AddAttributeError(
		attrPath,
		summary,
		summary+": "+err.Error(),
	)
	return diags
}

func getIDFromObject(
	ctx context.Context, data types.Object, idPath path.Path,
) (string, diag.Diagnostics) {
	pstr, diags := getPStringFromObject(ctx, data, constants.FieldID, idPath)
	if pstr == nil || *pstr == "" {
		diags.AddAttributeError(
			idPath.AtName(constants.FieldID),
			"id not set",
			"id is not set but is required",
		)
		return "", diags
	}
	return *pstr, diags
}

func metadataFromTF(
	ctx context.Context, data types.Object, nameMap map[string]map[string]string,
) (*common.ResourceMetadata, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	metadata := &common.ResourceMetadata{}
	_, innerDiag := conversion.MessageFromTFPath(ctx, data, metadata, path.Empty(), nameMap)
	diags.Append(innerDiag...)
	if innerDiag.HasError() {
		return nil, diags
	}

	metadataVal, ok := data.Attributes()[constants.FieldMetadata]
	if !ok {
		diags.AddAttributeError(
			path.Root(constants.FieldMetadata),
			"metadata not found",
			"metadata not found in the data object",
		)
		return nil, diags
	}
	mdObj, ok := metadataVal.(types.Object)
	if !ok {
		diags.AddAttributeError(
			path.Root(constants.FieldMetadata),
			"metadata not an object",
			"metadata has to be an object but is not",
		)
		return nil, diags
	}
	metadata2 := &common.ResourceMetadata{}
	if isKnown(mdObj) {
		_, innerDiag := conversion.MessageFromTFPath(
			ctx, mdObj, metadata2, path.Root(constants.FieldMetadata), nameMap,
		)
		diags.Append(innerDiag...)
		if innerDiag.HasError() {
			return nil, diags
		}
	}
	for _, fieldName := range constants.MetadataUnwrapped {
		fieldPath := mask.NewFieldPath(mask.FieldKey(fieldName))
		val, _, err := protobuf.GetAtFieldPath(metadata, fieldPath)
		if err != nil {
			if !errors.Is(err, protobuf.ErrNotFound) { // fields like hidden_labels
				diags.AddAttributeError(
					path.Root(constants.FieldMetadata),
					"get by fieldpath",
					fmt.Sprintf(
						"failed to get %q by fieldpath from metadata, %s",
						fieldName, err.Error(),
					),
				)
			}
			continue
		}
		err = protobuf.ReplaceAtFieldPath(metadata2, fieldPath, val)
		if err != nil {
			diags.AddAttributeError(
				path.Root(constants.FieldMetadata),
				"replace by fieldpath",
				fmt.Sprintf(
					"failed to replace %q by fieldpath from metadata, %s",
					fieldName, err.Error(),
				),
			)
			continue
		}
	}
	return metadata2, diags
}

func convertToObject(
	ctx context.Context,
	metadata, spec, status proto.Message,
	data types.Object,
	nameMap map[string]map[string]string,
) (types.Object, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	var innerDiag diag.Diagnostics
	attrTypes := data.AttributeTypes(ctx)
	if metadata != nil {
		attrs := data.Attributes()
		mdAttr, ok := attrs[constants.FieldMetadata]
		if ok {
			mdAttr, innerDiag, ok = conversion.MessageValueToTFRecursive(
				ctx, mdAttr, metadata, constants.FieldMetadata,
				nameMap,
			)
			diags.Append(innerDiag...)
			if ok {
				attrs[constants.FieldMetadata] = mdAttr
			}
		} else {
			diags.AddError("metadata not found",
				"metadata not found in the state object but was required",
			)
		}

		tempAttrs := map[string]attr.Value{}
		tempTypes := map[string]attr.Type{}
		for _, fieldName := range constants.MetadataUnwrapped {
			mdAttr, ok1 := attrs[string(fieldName)]
			mdType, ok2 := attrTypes[string(fieldName)]
			if !ok1 || !ok2 {
				diags.AddError("field not found",
					fmt.Sprintf(
						"%q not found in the state object but was required",
						fieldName,
					),
				)
				continue
			}
			tempAttrs[string(fieldName)] = mdAttr
			tempTypes[string(fieldName)] = mdType
		}
		tmpObj, innerDiag := basetypes.NewObjectValue(tempTypes, tempAttrs)
		diags.Append(innerDiag...)
		tmpObjvalue, innerDiag := conversion.MessageToTF(ctx, metadata, tmpObj, nameMap)
		diags.Append(innerDiag...)
		tmpObj, innerDiag = tmpObjvalue.ToObjectValue(ctx)
		diags.Append(innerDiag...)
		tflog.Debug(ctx, "some metadatas", map[string]interface{}{
			"mdsource": fmt.Sprint(metadata),
			"md1":      fmt.Sprint(mdAttr),
			"md2":      fmt.Sprint(tmpObj),
		})
		for _, fieldName := range constants.MetadataUnwrapped {
			if mdAttr, ok := tmpObj.Attributes()[string(fieldName)]; ok {
				attrs[string(fieldName)] = mdAttr
			}
		}
		data, innerDiag = basetypes.NewObjectValue(attrTypes, attrs)
		diags.Append(innerDiag...)
	}
	if spec != nil {
		tmpObjvalue, innerDiag := conversion.MessageToTF(ctx, spec, data, nameMap)
		diags.Append(innerDiag...)
		tmpObj, innerDiag := tmpObjvalue.ToObjectValue(ctx)
		diags.Append(innerDiag...)
		data = tmpObj
	}
	if status != nil {
		attrs := data.Attributes()
		statusAttr, ok := attrs[constants.FieldStatus]
		if !ok {
			diags.AddError("status not found",
				"status not found in the state object but was required",
			)
			return data, diags
		}
		statusAttr, innerDiag, ok = conversion.MessageValueToTFRecursive(
			ctx, statusAttr, status, constants.FieldStatus, nameMap,
		)
		diags.Append(innerDiag...)
		if ok {
			attrs[constants.FieldStatus] = statusAttr
		}

		data, innerDiag = basetypes.NewObjectValue(attrTypes, attrs)
		diags.Append(innerDiag...)
	}
	return data, diags
}
