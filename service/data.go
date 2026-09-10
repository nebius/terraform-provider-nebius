package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/nebius/gosdk/constants"
	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/gosdk/proto/fieldmask/protobuf"
	common "github.com/nebius/gosdk/proto/nebius/common/v1"
	"github.com/nebius/gosdk/serviceerror"
	"github.com/nebius/terraform-provider-nebius/conversion"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/provider"
	"github.com/nebius/terraform-provider-nebius/service/requestcontext"
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

func getOptionalPStringFromObject(ctx context.Context, data types.Object, name string, root path.Path) (
	*string, diag.Diagnostics,
) {
	if _, ok := data.Attributes()[name]; !ok {
		return nil, nil
	}

	return getPStringFromObject(ctx, data, name, root)
}

func ErrorToDiag(
	diags diag.Diagnostics,
	err error,
	summary string,
) diag.Diagnostics {
	if err == nil {
		return diags
	}
	formattableDiags, remainingErr := requestcontext.ExtractFormattableDiagnostics(err, summary)
	if len(formattableDiags) > 0 {
		diags.Append(formattableDiags...)
		if remainingErr == nil {
			return diags
		}
		err = remainingErr
	}
	if _, isServiceError := errors.AsType[*serviceerror.Error](err); isServiceError {
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

func metadataUnwrappedFields(nameMap map[string]map[string]string) []protoreflect.Name {
	fields := slices.Clone(constants.MetadataUnwrapped)
	if metadataFields, ok := nameMap[constants.MetadataMessageFullName]; ok {
		if metadataFields[constants.FieldRegion] == constants.FieldRegion &&
			!slices.Contains(fields, protoreflect.Name(constants.FieldRegion)) {
			fields = append(fields, constants.FieldRegion)
		}
	}
	return fields
}

func metadataFromTF(
	ctx context.Context, data types.Object, nameMap map[string]map[string]string,
) (*common.ResourceMetadata, diag.Diagnostics) {
	metadata, _, diags := metadataFromTFWithUnknowns(ctx, data, nameMap)
	return metadata, diags
}

func metadataFromTFWithUnknowns(
	ctx context.Context, data types.Object, nameMap map[string]map[string]string,
) (*common.ResourceMetadata, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	metadata := &common.ResourceMetadata{}
	unknowns, innerDiag := conversion.MessageFromTFPath(
		ctx, data, metadata, path.Empty(), nameMap,
	)
	diags.Append(innerDiag...)
	if innerDiag.HasError() {
		return nil, nil, diags
	}

	metadataVal, ok := data.Attributes()[constants.FieldMetadata]
	if !ok {
		diags.AddAttributeError(
			path.Root(constants.FieldMetadata),
			"metadata not found",
			"metadata not found in the data object",
		)
		return nil, nil, diags
	}
	mdObj, ok := metadataVal.(types.Object)
	if !ok {
		diags.AddAttributeError(
			path.Root(constants.FieldMetadata),
			"metadata not an object",
			"metadata has to be an object but is not",
		)
		return nil, nil, diags
	}
	metadata2 := &common.ResourceMetadata{}
	if isKnown(mdObj) {
		innerUnknowns, innerDiag := conversion.MessageFromTFPath(
			ctx, mdObj, metadata2, path.Root(constants.FieldMetadata), nameMap,
		)
		unknowns = ctypes.AppendUnknownMask(unknowns, mask.FieldPath{}, innerUnknowns)
		diags.Append(innerDiag...)
		if innerDiag.HasError() {
			return nil, nil, diags
		}
	} else if mdObj.IsUnknown() {
		metadataFields := metadata.ProtoReflect().Descriptor().Fields()
		for tfName := range mdObj.AttributeTypes(ctx) {
			protoName := protoreflect.Name(tfName)
			if aliases := nameMap[constants.MetadataMessageFullName]; aliases != nil {
				if alias, ok := aliases[tfName]; ok {
					protoName = protoreflect.Name(alias)
				}
			}
			if metadataFields.ByName(protoName) == nil {
				continue
			}
			unknowns = ctypes.AppendUnknownPath(
				unknowns,
				mask.NewFieldPath(mask.FieldKey(protoName)),
			)
		}
	}
	for _, fieldName := range metadataUnwrappedFields(nameMap) {
		if fieldName == constants.FieldRegion {
			if _, ok := data.Attributes()[string(fieldName)]; !ok {
				continue
			}
		}
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
	return metadata2, unknowns, diags
}

func objectWithEffectiveLabels(
	ctx context.Context,
	data types.Object,
) (types.Object, diag.Diagnostics) {
	attrs := data.Attributes()
	labelsAll, hasLabelsAll := attrs[constants.FieldLabelsAll]
	if !hasLabelsAll {
		return data, nil
	}
	if _, hasLabels := attrs[constants.FieldLabels]; !hasLabels {
		return data, nil
	}
	labelsAllMap, ok := labelsAll.(types.Map)
	if !ok {
		diags := diag.Diagnostics{}
		diags.AddAttributeError(
			path.Root(constants.FieldLabelsAll),
			"labels_all is not a map",
			fmt.Sprintf("Expected labels_all to be a map, got %s.", labelsAll.Type(ctx)),
		)
		return data, diags
	}
	diags := provider.ValidateLabels(
		labelsAllMap,
		path.Root(constants.FieldLabelsAll),
		true,
	)
	if diags.HasError() {
		return data, diags
	}

	cloned := maps.Clone(attrs)
	delete(cloned, constants.FieldLabelsAll)
	cloned[constants.FieldLabels] = labelsAll
	clonedTypes := maps.Clone(data.AttributeTypes(ctx))
	delete(clonedTypes, constants.FieldLabelsAll)
	ret, innerDiags := basetypes.NewObjectValue(clonedTypes, cloned)
	diags.Append(innerDiags...)
	return ret, diags
}

func requestMessagesFromTF(
	ctx context.Context,
	data types.Object,
	spec proto.Message,
	nameMap map[string]map[string]string,
) (*common.ResourceMetadata, diag.Diagnostics) {
	metadata, _, diags := requestMessagesFromTFWithUnknowns(ctx, data, spec, nameMap)
	return metadata, diags
}

func requestMessagesFromTFWithUnknowns(
	ctx context.Context,
	data types.Object,
	spec proto.Message,
	nameMap map[string]map[string]string,
) (*common.ResourceMetadata, *mask.Mask, diag.Diagnostics) {
	requestData, diags := objectWithEffectiveLabels(ctx, data)
	if diags.HasError() {
		return nil, nil, diags
	}
	metadata, metadataUnknowns, innerDiags := metadataFromTFWithUnknowns(
		ctx, requestData, nameMap,
	)
	diags.Append(innerDiags...)
	if innerDiags.HasError() {
		return nil, nil, diags
	}
	specUnknowns, innerDiags := conversion.MessageFromTF(ctx, requestData, spec, nameMap)
	diags.Append(innerDiags...)

	unknowns := ctypes.AppendUnknownMask(
		nil,
		mask.NewFieldPath(constants.FieldMetadata),
		metadataUnknowns,
	)
	unknowns = ctypes.AppendUnknownMask(
		unknowns,
		mask.NewFieldPath(constants.FieldSpec),
		specUnknowns,
	)
	return metadata, unknowns, diags
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
		resourceLabels, hasLabels := attrs[constants.FieldLabels]
		_, hasLabelsAll := attrTypes[constants.FieldLabelsAll]
		hasLabelsAll = hasLabelsAll && hasLabels
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

		unwrappedFields := metadataUnwrappedFields(nameMap)
		tempAttrs := map[string]attr.Value{}
		tempTypes := map[string]attr.Type{}
		for _, fieldName := range unwrappedFields {
			fieldAttr, ok1 := attrs[string(fieldName)]
			mdType, ok2 := attrTypes[string(fieldName)]
			if !ok1 || !ok2 {
				if fieldName == constants.FieldRegion {
					continue
				}
				diags.AddError("field not found",
					fmt.Sprintf(
						"%q not found in the state object but was required",
						fieldName,
					),
				)
				continue
			}
			tempAttrs[string(fieldName)] = fieldAttr
			tempTypes[string(fieldName)] = mdType
		}
		tmpObj, metadataDiags := basetypes.NewObjectValue(tempTypes, tempAttrs)
		diags.Append(metadataDiags...)
		tmpObjvalue, metadataDiags := conversion.MessageToTF(ctx, metadata, tmpObj, nameMap)
		diags.Append(metadataDiags...)
		tmpObj, metadataDiags = tmpObjvalue.ToObjectValue(ctx)
		diags.Append(metadataDiags...)
		for _, fieldName := range unwrappedFields {
			if mdAttr, ok := tmpObj.Attributes()[string(fieldName)]; ok {
				attrs[string(fieldName)] = mdAttr
			}
		}
		if hasLabelsAll {
			effectiveLabels := attrs[constants.FieldLabels]
			if effectiveLabels.IsNull() {
				effectiveLabels = types.MapValueMust(
					types.StringType,
					map[string]attr.Value{},
				)
			}
			attrs[constants.FieldLabelsAll] = effectiveLabels
			attrs[constants.FieldLabels] = resourceLabels
		}
		data, innerDiag = basetypes.NewObjectValue(attrTypes, attrs)
		diags.Append(innerDiag...)
	}
	if spec != nil {
		tmpObjvalue, specDiags := conversion.MessageToTF(ctx, spec, data, nameMap)
		diags.Append(specDiags...)
		tmpObj, specDiags := tmpObjvalue.ToObjectValue(ctx)
		diags.Append(specDiags...)
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
