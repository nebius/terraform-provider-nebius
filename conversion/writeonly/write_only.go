package writeonly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/terraform-provider-nebius/conversion"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
)

func pathToWOPath(p path.Path) path.Path {
	ret := path.Root(FieldName)
	for _, s := range p.Steps() {
		switch typed := s.(type) {
		case path.PathStepAttributeName:
			ret = ret.AtName(string(typed))
		case path.PathStepElementKeyInt:
			ret = ret.AtListIndex(int(typed))
		case path.PathStepElementKeyString:
			ret = ret.AtMapKey(string(typed))
		}
	}
	return ret
}

func ParseWriteOnlyFields(
	ctx context.Context,
	data *types.Object,
	dataMirror types.Object,
	writeOnlyMask *mask.Mask,
	pathPrefix mask.FieldPath,
	tfPathPrefix path.Path,
) (*types.Object, *mask.Mask, *mask.Mask, *mask.Mask, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	var unk *mask.Mask
	var dataUnk *mask.Mask
	var supplied *mask.Mask

	if dataMirror.IsNull() {
		return data, unk, dataUnk, supplied, diags
	}
	if dataMirror.IsUnknown() {
		writeOnlyFields, err := writeOnlyMask.GetSubMaskByPath(pathPrefix)
		if err != nil {
			diags.AddAttributeError(
				pathToWOPath(tfPathPrefix),
				"invalid write-only field mask",
				err.Error(),
			)
			return data, unk, dataUnk, supplied, diags
		}
		unk, err = unknownWriteOnlyFields(data, writeOnlyFields)
		if err != nil {
			diags.AddAttributeError(
				pathToWOPath(tfPathPrefix),
				"invalid write-only field mask",
				err.Error(),
			)
		}
		supplied = unk
		return data, unk, dataUnk, supplied, diags
	}
	stateAttributes := map[string]attr.Value{}
	stateTypes := map[string]attr.Type{}
	if data != nil {
		stateAttributes = data.Attributes()
		stateTypes = data.AttributeTypes(ctx)
	}
	dmAttrs := dataMirror.Attributes()
	for key, value := range dmAttrs {
		if key == VersionField && len(tfPathPrefix.Steps()) == 0 {
			continue
		}
		attrPath := pathPrefix.Join(mask.FieldKey(key))
		attrTfPath := tfPathPrefix.AtName(key)
		if value.IsNull() {
			continue
		}
		var innerAttr attr.Value = nil
		var innerType attr.Type = nil
		if data != nil {
			innerAttr = stateAttributes[key]
			innerTypeX, have := stateTypes[key]
			if !have {
				diags.AddAttributeError(
					attrTfPath,
					"missing attribute",
					fmt.Sprintf(
						"attribute %s for write-only %s not found in state",
						attrTfPath, pathToWOPath(attrTfPath),
					),
				)
				continue
			}
			innerType = innerTypeX
			if innerAttr != nil && innerAttr.IsUnknown() {
				dataUnk = ctypes.AppendUnknownPath(dataUnk, mask.FieldPath{mask.FieldKey(key)})
				innerAttr = nil
			}
		}
		isWriteOnly := attrPath.MatchesResetMaskFinal(writeOnlyMask)
		if value.IsUnknown() {
			if isWriteOnly {
				if innerAttr != nil && !innerAttr.IsNull() {
					continue
				}
				unk = ctypes.AppendUnknownPath(unk, mask.FieldPath{mask.FieldKey(key)})
				supplied = ctypes.AppendUnknownPath(supplied, mask.FieldPath{mask.FieldKey(key)})
				continue
			}
			writeOnlyFields, err := writeOnlyMask.GetSubMaskByPath(attrPath)
			if err != nil {
				diags.AddAttributeError(
					pathToWOPath(attrTfPath),
					"invalid write-only field mask",
					err.Error(),
				)
				continue
			}
			// Object fields have a fixed shape. Unknown lists and maps may add
			// elements, so their write-only descendants must remain unknown.
			var innerData *types.Object
			if object, ok := innerAttr.(types.Object); ok {
				innerData = &object
			}
			unknownFields, err := unknownWriteOnlyFields(innerData, writeOnlyFields)
			if err != nil {
				diags.AddAttributeError(
					pathToWOPath(attrTfPath),
					"invalid write-only field mask",
					err.Error(),
				)
				continue
			}
			unk = ctypes.AppendUnknownMask(unk, mask.FieldPath{mask.FieldKey(key)}, unknownFields)
			supplied = ctypes.AppendUnknownMask(supplied, mask.FieldPath{mask.FieldKey(key)}, unknownFields)
			continue
		}
		if isWriteOnly {
			if innerAttr != nil && !innerAttr.IsNull() {
				// ignore write-only value if state value is set
				continue
			}
			if innerType != nil {
				if !innerType.Equal(value.Type(ctx)) {
					diags.AddAttributeError(
						pathToWOPath(attrTfPath),
						"type mismatch",
						fmt.Sprintf(
							"write-only field %s type %s doesn't match the attribute %s type in the state %s",
							pathToWOPath(attrTfPath),
							value.Type(ctx).String(),
							attrTfPath,
							innerType.String(),
						),
					)
				}
			}
			stateAttributes[key] = value
			supplied = ctypes.AppendUnknownPath(supplied, mask.FieldPath{mask.FieldKey(key)})
			continue
		}
		switch typed := value.(type) {
		case types.Object:
			var innerData *types.Object
			if innerAttr == nil && innerType != nil {
				innerNull, innerDiag := conversion.NullOfType(
					ctx, innerType,
				)
				diags.Append(innerDiag...)
				if innerDiag.HasError() {
					continue
				}
				innerAttr = innerNull
			}
			if innerAttr != nil {
				innerDataNP, ok := innerAttr.(types.Object)
				if !ok {
					diags.AddAttributeError(
						attrTfPath,
						"invalid attribute type",
						fmt.Sprintf(
							"attribute %s is not an object, object expected by write-only mirror %s",
							attrTfPath,
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				innerData = &innerDataNP
			}
			innerData, innerUnk, innerDataUnk, innerSupplied, innerDiags := ParseWriteOnlyFields(
				ctx, innerData, typed,
				writeOnlyMask, attrPath, attrTfPath,
			)
			unk = ctypes.AppendUnknownMask(unk, mask.FieldPath{mask.FieldKey(key)}, innerUnk)
			dataUnk = ctypes.AppendUnknownMask(dataUnk, mask.FieldPath{mask.FieldKey(key)}, innerDataUnk)
			supplied = ctypes.AppendUnknownMask(supplied, mask.FieldPath{mask.FieldKey(key)}, innerSupplied)
			diags.Append(innerDiags...)
			if innerData != nil {
				innerAttr = *innerData
			}
		case types.List:
			if len(typed.Elements()) == 0 {
				continue
			}
			dataElements := []attr.Value{}
			var listElementType attr.Type
			if innerAttr == nil && innerType != nil {
				listType, ok := innerType.(types.ListType)
				if !ok {
					diags.AddAttributeError(
						pathToWOPath(attrTfPath),
						"invalid attribute type",
						fmt.Sprintf(
							"attribute %s is not a list, list expected by write-only mirror %s",
							attrTfPath,
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				emptyList, innerDiag := types.ListValue(listType.ElementType(), []attr.Value{})
				diags.Append(innerDiag...)
				if innerDiag.HasError() {
					continue
				}
				innerAttr = emptyList
			}
			if innerAttr != nil {
				innerListNP, ok := innerAttr.(types.List)
				if !ok {
					diags.AddAttributeError(
						pathToWOPath(attrTfPath),
						"invalid attribute type",
						fmt.Sprintf(
							"attribute %s is not a list, list expected by write-only mirror %s",
							attrTfPath,
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				if !innerListNP.IsNull() && len(innerListNP.Elements()) != 0 {
					dataElements = innerListNP.Elements()
					if len(dataElements) != len(typed.Elements()) {
						diags.AddAttributeError(
							pathToWOPath(attrTfPath),
							"list length mismatch",
							fmt.Sprintf(
								"attribute %s has %d elements, but write-only field %s has %d elements, match the lists by adding empty objects where necessary",
								attrTfPath, len(dataElements),
								pathToWOPath(attrTfPath), len(typed.Elements()),
							),
						)
					}
				}
				listElementType = innerListNP.ElementType(ctx)
			}

			for i, el := range typed.Elements() {
				elementPath := attrPath.Join(mask.FieldKey(fmt.Sprintf("%d", i)))
				elementTfPath := attrTfPath.AtListIndex(i)
				var dataElement *types.Object
				if len(dataElements) > i {
					dataElementObj := dataElements[i]
					dataElementNP, ok := dataElementObj.(types.Object)
					if !ok {
						diags.AddAttributeError(
							elementTfPath,
							"invalid list element",
							fmt.Sprintf(
								"element %s of state list is not an object, while write-only list element %s is",
								elementTfPath,
								pathToWOPath(attrTfPath),
							),
						)
						continue
					}
					dataElement = &dataElementNP
				}
				if dataElement != nil && dataElement.IsUnknown() {
					dataUnk = ctypes.AppendUnknownPath(
						dataUnk,
						mask.FieldPath{
							mask.FieldKey(key),
							mask.FieldKey(fmt.Sprintf("%d", i)),
						},
					)
					dataElement = nil
				}
				if dataElement == nil && innerAttr != nil {
					emptyObj, innerDiag := conversion.NullOfType(
						ctx, listElementType,
					)
					diags.Append(innerDiag...)
					if innerDiag.HasError() {
						continue
					}
					dataElementNP, ok := emptyObj.(types.Object)
					if !ok {
						diags.AddAttributeError(
							elementTfPath,
							"invalid list element",
							fmt.Sprintf(
								"element %s of state list is not an object, while write-only list element %s is",
								elementTfPath,
								pathToWOPath(attrTfPath),
							),
						)
						continue
					}
					dataElement = &dataElementNP
				}
				obj, ok := el.(types.Object)
				if !ok {
					diags.AddAttributeError(
						pathToWOPath(elementTfPath),
						"invalid list element",
						fmt.Sprintf(
							"write-only list element %s is not an object",
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				dataElement, innerUnk, innerDataUnk, innerSupplied, innerDiag := ParseWriteOnlyFields(
					ctx, dataElement, obj,
					writeOnlyMask, elementPath, elementTfPath,
				)
				unk = ctypes.AppendUnknownMask(
					unk,
					mask.FieldPath{mask.FieldKey(key), mask.FieldKey(fmt.Sprintf("%d", i))},
					innerUnk,
				)
				dataUnk = ctypes.AppendUnknownMask(
					dataUnk,
					mask.FieldPath{mask.FieldKey(key), mask.FieldKey(fmt.Sprintf("%d", i))},
					innerDataUnk,
				)
				supplied = ctypes.AppendUnknownMask(
					supplied,
					mask.FieldPath{mask.FieldKey(key), mask.FieldKey(fmt.Sprintf("%d", i))},
					innerSupplied,
				)
				diags.Append(innerDiag...)
				if dataElement != nil {
					if len(dataElements) > i {
						dataElements[i] = *dataElement
					} else {
						dataElements = append(dataElements, *dataElement)
					}
				}
			}
			if innerAttr != nil && len(dataElements) != 0 {
				innerAttrX, innerDiag := types.ListValue(
					listElementType,
					dataElements,
				)
				diags.Append(innerDiag...)
				innerAttr = innerAttrX
			}
		case types.Map:
			if len(typed.Elements()) == 0 {
				continue
			}
			dataElements := map[string]attr.Value{}
			var mapElementType attr.Type
			if innerAttr == nil && innerType != nil {
				mapType, ok := innerType.(types.MapType)
				if !ok {
					diags.AddAttributeError(
						pathToWOPath(attrTfPath),
						"invalid attribute type",
						fmt.Sprintf(
							"attribute %s is not a map, map expected by write-only mirror %s",
							attrTfPath,
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				emptyMap, innerDiag := types.MapValue(
					mapType.ElementType(), map[string]attr.Value{},
				)
				diags.Append(innerDiag...)
				if innerDiag.HasError() {
					continue
				}
				innerAttr = emptyMap
			}
			if innerAttr != nil {
				innerMapNP, ok := innerAttr.(types.Map)
				if !ok {
					diags.AddAttributeError(
						pathToWOPath(attrTfPath),
						"invalid attribute type",
						fmt.Sprintf(
							"attribute %s is not a map, map expected by write-only mirror %s",
							attrTfPath,
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				if !innerMapNP.IsNull() && len(innerMapNP.Elements()) != 0 {
					dataElements = innerMapNP.Elements()
				}
				if dataElements == nil {
					dataElements = map[string]attr.Value{}
				}
				mapElementType = innerMapNP.ElementType(ctx)
			}
			for mapKey, el := range typed.Elements() {
				elementPath := attrPath.Join(mask.FieldKey(mapKey))
				elementTfPath := attrTfPath.AtMapKey(mapKey)
				var dataElement *types.Object
				if dataElementVal, ok := dataElements[mapKey]; ok {
					dataElementNP, ok := dataElementVal.(types.Object)
					if !ok {
						diags.AddAttributeError(
							elementTfPath,
							"invalid map element",
							fmt.Sprintf(
								"element %s of state map is not an object while write-only map element %s is",
								elementTfPath,
								pathToWOPath(attrTfPath),
							),
						)
						continue
					}
					dataElement = &dataElementNP
				}
				if dataElement != nil && dataElement.IsUnknown() {
					dataUnk = ctypes.AppendUnknownPath(
						dataUnk,
						mask.FieldPath{
							mask.FieldKey(key),
							mask.FieldKey(mapKey),
						},
					)
					dataElement = nil
				}
				if dataElement == nil && innerAttr != nil {
					emptyObj, innerDiag := conversion.NullOfType(
						ctx, mapElementType,
					)
					diags.Append(innerDiag...)
					if innerDiag.HasError() {
						continue
					}
					dataElementNP, ok := emptyObj.(types.Object)
					if !ok {
						diags.AddAttributeError(
							elementTfPath,
							"invalid map element",
							fmt.Sprintf(
								"element %s of state map is not an object while write-only map element %s is",
								elementTfPath,
								pathToWOPath(attrTfPath),
							),
						)
						continue
					}
					dataElement = &dataElementNP
				}
				obj, ok := el.(types.Object)
				if !ok {
					diags.AddAttributeError(
						pathToWOPath(elementTfPath),
						"invalid map element",
						fmt.Sprintf(
							"element of write-only map %s is not an object",
							pathToWOPath(attrTfPath),
						),
					)
					continue
				}
				dataElement, innerUnk, innerDataUnk, innerSupplied, innerDiags := ParseWriteOnlyFields(
					ctx, dataElement, obj,
					writeOnlyMask, elementPath, elementTfPath,
				)
				unk = ctypes.AppendUnknownMask(
					unk,
					mask.FieldPath{mask.FieldKey(key), mask.FieldKey(mapKey)},
					innerUnk,
				)
				dataUnk = ctypes.AppendUnknownMask(
					dataUnk,
					mask.FieldPath{mask.FieldKey(key), mask.FieldKey(mapKey)},
					innerDataUnk,
				)
				supplied = ctypes.AppendUnknownMask(
					supplied,
					mask.FieldPath{mask.FieldKey(key), mask.FieldKey(mapKey)},
					innerSupplied,
				)
				diags.Append(innerDiags...)
				if dataElement != nil {
					dataElements[mapKey] = *dataElement
				}
			}
			if innerType != nil {
				innerAttrX, innerDiag := types.MapValue(
					mapElementType,
					dataElements,
				)
				diags.Append(innerDiag...)
				innerAttr = innerAttrX
			}
		default:
			diags.AddAttributeError(
				pathToWOPath(attrTfPath),
				"unsupported intermediate type",
				fmt.Sprintf(
					"unsupported intermediate type %T of write-only attribute %s, should be object, list or map",
					typed,
					pathToWOPath(attrTfPath),
				),
			)
		}
		stateAttributes[key] = innerAttr
	}
	if data != nil {
		for key, val := range stateTypes {
			if _, ok := stateAttributes[key]; !ok {
				newNull, innerDiag := conversion.NullOfType(
					ctx, val,
				)
				diags.Append(innerDiag...)
				if innerDiag.HasError() {
					continue
				}
				stateAttributes[key] = newNull
			}
		}
		newObj, innerDiag := types.ObjectValue(
			stateTypes, stateAttributes,
		)
		diags.Append(innerDiag...)
		if innerDiag.HasError() {
			diags.AddError(
				"failed to create new object",
				fmt.Sprintf(
					"failed to create new object %s from attributes with secrets",
					tfPathPrefix,
				),
			)
			return data, unk, dataUnk, supplied, diags
		}
		data = &newObj
	}

	return data, unk, dataUnk, supplied, diags
}

// unknownWriteOnlyFields returns write-only fields that an unknown mirror may
// still supply. A known main value takes precedence over its mirror value.
func unknownWriteOnlyFields(
	data *types.Object,
	writeOnlyFields *mask.Mask,
) (*mask.Mask, error) {
	if writeOnlyFields == nil {
		return nil, nil
	}
	unknown, err := writeOnlyFields.Copy()
	if err != nil {
		return nil, fmt.Errorf("copy write-only fields: %w", err)
	}
	if data == nil || data.IsNull() || data.IsUnknown() {
		return unknown, nil
	}

	known := knownMainWriteOnlyFields(data, writeOnlyFields)
	if !known.IsEmpty() {
		if err := unknown.SubtractResetMask(known); err != nil {
			return nil, fmt.Errorf("exclude known main fields: %w", err)
		}
	}
	if unknown.IsEmpty() {
		return nil, nil
	}
	return unknown, nil
}

func knownMainWriteOnlyFields(
	data *types.Object,
	writeOnlyFields *mask.Mask,
) *mask.Mask {
	known := mask.New()
	attributes := data.Attributes()
	for key, child := range writeOnlyFields.FieldParts {
		if child == nil {
			continue
		}
		value := attributes[string(key)]
		if value == nil || value.IsNull() || value.IsUnknown() {
			continue
		}
		if child.IsEmpty() {
			known.FieldParts[key] = mask.New()
			continue
		}
		object, ok := value.(types.Object)
		if !ok {
			continue
		}
		knownChild := knownMainWriteOnlyFields(&object, child)
		if !knownChild.IsEmpty() {
			known.FieldParts[key] = knownChild
		}
	}
	return known
}
