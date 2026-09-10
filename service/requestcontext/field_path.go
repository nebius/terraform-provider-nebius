package requestcontext

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
)

// FieldPathToTFPath maps a Nebius field path rooted at metadata or spec to a
// Terraform attribute path.
func FieldPathToTFPath(
	fieldPath string,
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) (path.Path, error) {
	fp, err := parseFieldPath(fieldPath)
	if err != nil {
		return path.Path{}, err
	}
	if schemaType == nil {
		return path.Path{}, fmt.Errorf("schema type is required")
	}
	tfPath := path.Empty()
	currentType := schemaType
	for i := 0; i < len(fp); i++ {
		key := fp[i]
		keyStr := string(key)
		switch typed := currentType.(type) {
		case attr.TypeWithAttributeTypes:
			attrs := typed.AttributeTypes()
			skipPrefix, err := skipRootPrefix(
				i,
				fp,
				keyStr,
				attrs,
				fieldNameMap,
			)
			if err != nil {
				return path.Path{}, err
			}
			if skipPrefix {
				i++
				key = fp[i]
				keyStr = string(key)
			}
			tfName, ok, err := fieldNameToTFName(
				keyStr,
				attrs,
				fieldNameMap,
			)
			if err != nil {
				return path.Path{}, err
			}
			if !ok {
				return path.Path{}, fmt.Errorf(
					"attribute %q is not found in the current schema object",
					keyStr,
				)
			}
			tfPath = tfPath.AtName(tfName)
			currentType = attrs[tfName]
			continue
		case types.ListType:
			idx, err := fieldKeyToIndex(key)
			if err != nil {
				return path.Path{}, err
			}
			tfPath = tfPath.AtListIndex(idx)
			currentType = typed.ElemType
			continue
		case types.SetType:
			// Nebius indexes repeated fields, but Terraform sets have no stable
			// index. Return the nearest addressable collection path.
			if _, err := fieldKeyToIndex(key); err != nil {
				return path.Path{}, err
			}
			return tfPath, nil
		case types.TupleType:
			idx, err := fieldKeyToIndex(key)
			if err != nil {
				return path.Path{}, err
			}
			tfPath = tfPath.AtListIndex(idx)
			if idx >= len(typed.ElemTypes) {
				return path.Path{}, fmt.Errorf(
					"tuple index %d is out of range for tuple with %d elements",
					idx,
					len(typed.ElemTypes),
				)
			}
			currentType = typed.ElemTypes[idx]
			continue
		case types.MapType:
			tfPath = tfPath.AtMapKey(keyStr)
			currentType = typed.ElemType
			continue
		default:
			if currentType == nil {
				return path.Path{}, fmt.Errorf(
					"schema type is not defined for path segment %q",
					keyStr,
				)
			}
			return path.Path{}, fmt.Errorf(
				"unsupported schema type %T at path segment %q",
				currentType,
				keyStr,
			)
		}
	}
	return tfPath, nil
}

func skipRootPrefix(
	i int,
	fp mask.FieldPath,
	keyStr string,
	rootAttrs map[string]attr.Type,
	fieldNameMap map[string]map[string]string,
) (bool, error) {
	if i != 0 || len(fp) < 2 {
		return false, nil
	}
	if keyStr != "spec" && keyStr != "metadata" {
		return false, nil
	}
	_, ok, err := fieldNameToTFName(
		string(fp[1]),
		rootAttrs,
		fieldNameMap,
	)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return true, nil
}

func fieldNameToTFName(
	fieldName string,
	attrs map[string]attr.Type,
	fieldNameMap map[string]map[string]string,
) (string, bool, error) {
	matchesByProto := map[string]struct{}{}
	var aliasProtoNames []string
	for _, parentAliases := range fieldNameMap {
		for tfName, protoName := range parentAliases {
			if _, ok := attrs[tfName]; !ok {
				continue
			}
			if protoName == fieldName {
				matchesByProto[tfName] = struct{}{}
				continue
			}
			if tfName == fieldName {
				aliasProtoNames = append(aliasProtoNames, protoName)
			}
		}
	}

	if len(aliasProtoNames) > 0 {
		sort.Strings(aliasProtoNames)
		return "", false, fmt.Errorf(
			"field path uses terraform alias %q, expected proto field name: %s",
			fieldName,
			strings.Join(aliasProtoNames, ", "),
		)
	}

	if len(matchesByProto) == 1 {
		for tfName := range matchesByProto {
			return tfName, true, nil
		}
	}
	if len(matchesByProto) > 1 {
		possible := make([]string, 0, len(matchesByProto))
		for tfName := range matchesByProto {
			possible = append(possible, tfName)
		}
		sort.Strings(possible)
		return "", false, fmt.Errorf(
			"field %q matches multiple terraform attributes by alias: %s",
			fieldName,
			strings.Join(possible, ", "),
		)
	}

	if _, ok := attrs[fieldName]; ok {
		return fieldName, true, nil
	}
	return "", false, nil
}

func fieldKeyToIndex(key mask.FieldKey) (int, error) {
	keyStr := string(key)
	idx, err := strconv.Atoi(keyStr)
	if err != nil {
		return 0, fmt.Errorf("expected list index, got %q", keyStr)
	}
	if idx < 0 {
		return 0, fmt.Errorf("negative list index: %d", idx)
	}
	return idx, nil
}

func parseFieldPath(fieldPath string) (mask.FieldPath, error) {
	parsedMask, err := mask.Parse(fieldPath)
	if err != nil {
		return nil, fmt.Errorf("unparsable mask: %w", err)
	}
	fp, err := parsedMask.ToFieldPath()
	if err != nil {
		return nil, fmt.Errorf("unparsable mask: %w", err)
	}
	if len(fp) == 0 {
		return nil, fmt.Errorf("empty field path")
	}
	return fp, nil
}
