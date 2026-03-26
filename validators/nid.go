package validators

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/proto"

	checknid "github.com/nebius/gosdk/check-nid"
	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/terraform-provider-nebius/conversion"
)

type nidValidator struct {
	allowedTypes []string
	summary      string
}

var _ validator.String = (*nidValidator)(nil)
var _ validator.Dynamic = (*nidValidator)(nil)
var _ validator.List = (*nidValidator)(nil)
var _ validator.Map = (*nidValidator)(nil)

func NIDValidator(allowedTypes []string) *nidValidator {
	return &nidValidator{
		allowedTypes: allowedTypes,
		summary:      "invalid Nebius ID",
	}
}

func ParentNIDValidator(allowedTypes []string) *nidValidator {
	return &nidValidator{
		allowedTypes: allowedTypes,
		summary:      "invalid Nebius parent ID",
	}
}

func (v *nidValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *nidValidator) MarkdownDescription(_ context.Context) string {
	if len(v.allowedTypes) == 0 {
		return "Validate value is a Nebius ID (warning only)."
	}
	nidsList := make([]string, len(v.allowedTypes))
	for i, t := range v.allowedTypes {
		nidsList[i] = t + "-e01abc"
	}
	return fmt.Sprintf("Validate value is a Nebius ID of the allowed resource types %s (warning only).", strings.Join(nidsList, ", "))
}

func (v *nidValidator) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *nidValidator) ValidateDynamic(
	ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *nidValidator) ValidateList(
	ctx context.Context, req validator.ListRequest, resp *validator.ListResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *nidValidator) ValidateMap(
	ctx context.Context, req validator.MapRequest, resp *validator.MapResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *nidValidator) validate(
	ctx context.Context, value attr.Value, currentPath path.Path,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	switch val := value.(type) {
	case basetypes.StringValuable:
		strVal, innerDiags := val.ToStringValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || strVal.IsNull() || strVal.IsUnknown() {
			return diags
		}
		v.addWarning(currentPath, strVal.ValueString(), currentPath, &diags)
	case basetypes.ListValuable:
		listVal, innerDiags := val.ToListValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || listVal.IsNull() || listVal.IsUnknown() {
			return diags
		}
		for i, el := range listVal.Elements() {
			diags.Append(v.validate(ctx, el, currentPath.AtListIndex(i))...)
		}
	case basetypes.MapValuable:
		mapVal, innerDiags := val.ToMapValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || mapVal.IsNull() || mapVal.IsUnknown() {
			return diags
		}
		keys := make([]string, 0, len(mapVal.Elements()))
		for k := range mapVal.Elements() {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			diags.Append(v.validate(ctx, mapVal.Elements()[k], currentPath.AtMapKey(k))...)
		}
	default:
		if value.IsNull() || value.IsUnknown() {
			return diags
		}
		diags.AddAttributeError(
			currentPath,
			"unsupported NID validator value",
			fmt.Sprintf(
				internalErrorClarification+
					"Unexpected value type %T at %s for NID validation",
				value, currentPath,
			),
		)
	}

	return diags
}

func (v *nidValidator) addWarning(
	targetPath path.Path,
	value string,
	callerPath path.Path,
	diags *diag.Diagnostics,
) {
	if warning := checknid.ValidateNIDString(value, v.allowedTypes); warning != "" {
		diags.AddAttributeWarning(
			targetPath,
			v.summary,
			fmt.Sprintf("Invalid Nebius ID for %s: %s", callerPath, warning),
		)
	}
}

type recursiveNIDValidator struct {
	newMessage  func() proto.Message
	nameMap     map[string]map[string]string
	nidCheckCtx *checknid.NIDCheckContext
}

var _ validator.Dynamic = (*recursiveNIDValidator)(nil)

type SubfieldSettings = checknid.SubfieldSettings

func RecursiveNIDValidator(
	template proto.Message,
	nameMap map[string]map[string]string,
	subfieldSettings []*SubfieldSettings,
) *recursiveNIDValidator {
	return &recursiveNIDValidator{
		newMessage: func() proto.Message {
			if template == nil {
				return nil
			}
			return proto.Clone(template)
		},
		nameMap:     nameMap,
		nidCheckCtx: checknid.NewNIDCheckContext(subfieldSettings),
	}
}

func (v *recursiveNIDValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *recursiveNIDValidator) MarkdownDescription(_ context.Context) string {
	return "Validate recursive dynamic field using NID annotations (warning only)."
}

func (v *recursiveNIDValidator) ValidateDynamic(
	ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *recursiveNIDValidator) validate(
	ctx context.Context, value attr.Value, currentPath path.Path,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	if value.IsNull() || value.IsUnknown() {
		return diags
	}

	switch val := value.(type) {
	case basetypes.DynamicValuable:
		diags.Append(v.validateAttr(ctx, val, currentPath)...)
	default:
		diags.AddAttributeError(
			currentPath,
			"unsupported recursive validator value",
			fmt.Sprintf(
				internalErrorClarification+
					"Unexpected value type %T at %s for recursive validation",
				value, currentPath,
			),
		)
	}

	return diags
}

func (v *recursiveNIDValidator) validateAttr(
	ctx context.Context,
	value attr.Value,
	currentPath path.Path,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	switch val := value.(type) {
	case basetypes.DynamicValuable:
		msg := v.newMessage()
		if msg == nil {
			diags.AddAttributeError(
				currentPath,
				"missing recursive message template",
				internalErrorClarification+"validator has no message template to validate value.",
			)
			return diags
		}
		unknowns, innerDiag := conversion.MessageFromDynamic(
			ctx, val, msg, v.nameMap,
		)
		diags.Append(innerDiag...)
		if innerDiag.HasError() {
			return diags
		}
		diags.Append(v.checkMessage(msg, unknowns, currentPath)...)
	default:
		diags.AddAttributeError(
			currentPath,
			"unsupported recursive message value",
			fmt.Sprintf(
				internalErrorClarification+
					"Unexpected value type %T at %s for recursive message conversion",
				value, currentPath,
			),
		)
	}

	return diags
}

func (v *recursiveNIDValidator) checkMessage(
	msg proto.Message,
	unknowns *mask.Mask,
	currentPath path.Path,
) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for relPath, warning := range checknid.CheckMessageFields(msg, v.nidCheckCtx) {
		if matchesUnknownPath(unknowns, relPath) {
			continue
		}
		diags.AddAttributeWarning(
			currentPath,
			"invalid Nebius ID",
			fmt.Sprintf("Invalid Nebius ID at %s: %s", relPath, warning),
		)
	}
	return diags
}

func matchesUnknownPath(unknowns *mask.Mask, relPath string) bool {
	fp, err := parseUnknownPath(relPath)
	if err != nil {
		return false
	}
	if unknowns == nil {
		return false
	}
	if unknowns.IsEmpty() {
		return true
	}
	return fp.MatchesResetMask(unknowns)
}

func parseUnknownPath(path string) (mask.FieldPath, error) {
	if path == "" {
		return mask.FieldPath{}, nil
	}

	ret := make(mask.FieldPath, 0, strings.Count(path, ".")+1)
	for i := 0; i < len(path); {
		for i < len(path) && path[i] == '.' {
			i++
		}
		if i >= len(path) {
			break
		}
		if path[i] == '[' {
			key, nextPos, err := parseUnknownSubscript(path, i)
			if err != nil {
				return nil, err
			}
			ret = append(ret, mask.FieldKey(key))
			i = nextPos
			continue
		}
		start := i
		for i < len(path) && path[i] != '.' && path[i] != '[' {
			i++
		}
		if start == i {
			return nil, strconv.ErrSyntax
		}
		segment := path[start:i]
		if len(segment) >= 2 && segment[0] == '"' && segment[len(segment)-1] == '"' {
			unquoted, err := strconv.Unquote(segment)
			if err == nil {
				segment = unquoted
			}
		}
		ret = append(ret, mask.FieldKey(segment))
	}
	return ret, nil
}

func parseUnknownSubscript(path string, pos int) (string, int, error) {
	if pos >= len(path) || path[pos] != '[' {
		return "", 0, strconv.ErrSyntax
	}
	if pos+1 >= len(path) {
		return "", 0, strconv.ErrSyntax
	}

	if path[pos+1] == '"' {
		endQuote := pos + 2
		for endQuote < len(path) {
			if path[endQuote] == '\\' {
				endQuote += 2
				continue
			}
			if path[endQuote] == '"' {
				break
			}
			endQuote++
		}
		if endQuote >= len(path) || path[endQuote] != '"' {
			return "", 0, strconv.ErrSyntax
		}
		if endQuote+1 >= len(path) || path[endQuote+1] != ']' {
			return "", 0, strconv.ErrSyntax
		}
		quoted := path[pos+1 : endQuote+1]
		key, err := strconv.Unquote(quoted)
		if err != nil {
			return "", 0, err
		}
		return key, endQuote + 2, nil
	}

	end := strings.IndexByte(path[pos+1:], ']')
	if end == -1 {
		return "", 0, strconv.ErrSyntax
	}
	end += pos + 1
	return path[pos+1 : end], end + 1, nil
}
