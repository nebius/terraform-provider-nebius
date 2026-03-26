package requestcontext

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/osteele/liquid"

	"github.com/nebius/gosdk/proto/fieldmask/mask"
	commonpb "github.com/nebius/gosdk/proto/nebius/common/v1"
)

func processWarnings(
	warnings *commonpb.Warnings,
	diags diag.Diagnostics,
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) diag.Diagnostics {
	if warnings == nil {
		return diags
	}
	engine := liquid.NewEngine()
	engine.RegisterFilter("field_name", fieldName(schemaType, fieldNameMap))
	bindings := liquid.Bindings{}
	for _, warning := range warnings.GetWarnings() {
		if warning == nil {
			continue
		}
		summaryTemplate, summaryFallback := normalizeWarningTemplate(
			warning.GetSummary(),
			warning.GetSummaryFallback(),
		)
		detailsTemplate, detailsFallback := normalizeWarningTemplate(
			warning.GetDetails(),
			warning.GetDetailsFallback(),
		)
		summary := applyWarningTemplate(
			engine, bindings, summaryTemplate, summaryFallback,
		)
		details := applyWarningTemplate(
			engine, bindings, detailsTemplate, detailsFallback,
		)
		attrPath, err := warningFieldPathToTFPath(
			warning.GetPath(),
			schemaType,
			fieldNameMap,
		)
		if err != nil {
			diags.AddWarning(summary, details)
			continue
		}
		diags.AddAttributeWarning(attrPath, summary, details)
	}
	return diags
}

func fieldName(
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) func(string) (string, error) {
	return func(fieldPath string) (string, error) {
		attrPath, err := warningFieldPathToTFPath(
			fieldPath,
			schemaType,
			fieldNameMap,
		)
		if err != nil {
			return "", err
		}
		return attrPath.String(), nil
	}
}

func normalizeWarningTemplate(template, fallback string) (string, string) {
	switch template {
	case "":
		return fallback, ""
	case fallback:
		return template, ""
	default:
		return template, fallback
	}
}

func applyWarningTemplate(
	engine *liquid.Engine,
	bindings liquid.Bindings,
	template string,
	fallback string,
) string {
	if template == "" {
		return fallback
	}
	tmpl, err := engine.ParseString(template)
	if err != nil {
		return fallback
	}
	res, err := tmpl.RenderString(bindings)
	if err != nil {
		return fallback
	}
	return res
}

func warningFieldPathToTFPath(
	fieldPath string,
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) (path.Path, error) {
	fp, err := parseWarningFieldPath(fieldPath)
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
			skipPrefix, err := skipWarningRootPrefix(
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
			tfName, ok, err := warningFieldNameToTFName(
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
			idx, err := warningFieldKeyToIndex(key)
			if err != nil {
				return path.Path{}, err
			}
			tfPath = tfPath.AtListIndex(idx)
			currentType = typed.ElemType
			continue
		case types.SetType:
			idx, err := warningFieldKeyToIndex(key)
			if err != nil {
				return path.Path{}, err
			}
			tfPath = tfPath.AtListIndex(idx)
			currentType = typed.ElemType
			continue
		case types.TupleType:
			idx, err := warningFieldKeyToIndex(key)
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

func skipWarningRootPrefix(
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
	_, ok, err := warningFieldNameToTFName(
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

func warningFieldNameToTFName(
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

func warningFieldKeyToIndex(key mask.FieldKey) (int, error) {
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

func parseWarningFieldPath(fieldPath string) (mask.FieldPath, error) {
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
