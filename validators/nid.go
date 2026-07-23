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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	checknid "github.com/nebius/gosdk/check-nid"
	"github.com/nebius/gosdk/nid"
	"github.com/nebius/gosdk/proto/fieldmask/mask"
	"github.com/nebius/gosdk/proto/nebius"
	"github.com/nebius/terraform-provider-nebius/conversion"
)

type nidValidator struct{}

var _ validator.String = (*nidValidator)(nil)
var _ validator.Dynamic = (*nidValidator)(nil)
var _ validator.List = (*nidValidator)(nil)
var _ validator.Map = (*nidValidator)(nil)

func NIDValidator() *nidValidator {
	return &nidValidator{}
}

// ParentNIDValidator is kept as an alias for generated code compatibility.
// NID types are intentionally not checked.
func ParentNIDValidator() *nidValidator {
	return NIDValidator()
}

func ListNIDValidator() validator.List {
	return NIDValidator()
}

func MapNIDValidator() validator.Map {
	return NIDValidator()
}

func (v *nidValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *nidValidator) MarkdownDescription(_ context.Context) string {
	return "Validate value has the Nebius ID format (warning only)."
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
		if warning := validateNIDFormat(strVal.ValueString()); warning != "" {
			diags.AddAttributeWarning(
				currentPath,
				"invalid Nebius ID",
				fmt.Sprintf("Invalid Nebius ID for %s: %s", currentPath, warning),
			)
		}
	case basetypes.ListValuable:
		listVal, innerDiags := val.ToListValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || listVal.IsNull() || listVal.IsUnknown() {
			return diags
		}
		for i, element := range listVal.Elements() {
			diags.Append(v.validate(ctx, element, currentPath.AtListIndex(i))...)
		}
	case basetypes.MapValuable:
		mapVal, innerDiags := val.ToMapValue(ctx)
		diags.Append(innerDiags...)
		if innerDiags.HasError() || mapVal.IsNull() || mapVal.IsUnknown() {
			return diags
		}
		keys := make([]string, 0, len(mapVal.Elements()))
		for key := range mapVal.Elements() {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			diags.Append(v.validate(ctx, mapVal.Elements()[key], currentPath.AtMapKey(key))...)
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

type recursiveNIDValidator struct {
	newMessage func() proto.Message
	nameMap    map[string]map[string]string
	nidPaths   []*checknid.SubfieldPath
}

var _ validator.Dynamic = (*recursiveNIDValidator)(nil)

type SubfieldSettings = checknid.SubfieldSettings

// RecursiveNIDValidator is kept for source compatibility with previously
// generated provider code. It validates format only.
func RecursiveNIDValidator(
	template proto.Message,
	nameMap map[string]map[string]string,
	subfieldSettings []*SubfieldSettings,
) *recursiveNIDValidator {
	paths := make([]string, 0, len(subfieldSettings))
	for _, setting := range subfieldSettings {
		if setting != nil && hasNIDResourceSetting(setting.GetNid()) {
			paths = append(paths, setting.GetFieldPath())
		}
	}
	return RecursiveNIDFormatValidator(template, nameMap, paths)
}

func RecursiveNIDFormatValidator(
	template proto.Message,
	nameMap map[string]map[string]string,
	nidPaths []string,
) *recursiveNIDValidator {
	return &recursiveNIDValidator{
		newMessage: func() proto.Message {
			if template == nil {
				return nil
			}
			return proto.Clone(template)
		},
		nameMap:  nameMap,
		nidPaths: compileNIDPaths(nidPaths),
	}
}

func (v *recursiveNIDValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v *recursiveNIDValidator) MarkdownDescription(_ context.Context) string {
	return "Validate recursive dynamic field using Nebius ID format annotations (warning only)."
}

func (v *recursiveNIDValidator) ValidateDynamic(
	ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse,
) {
	resp.Diagnostics.Append(v.validate(ctx, req.ConfigValue, req.Path)...)
}

func (v *recursiveNIDValidator) validate(
	ctx context.Context, value attr.Value, currentPath path.Path,
) diag.Diagnostics {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	dynamic, ok := value.(basetypes.DynamicValuable)
	if !ok {
		return diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
			currentPath,
			"unsupported recursive validator value",
			fmt.Sprintf(
				internalErrorClarification+
					"Unexpected value type %T at %s for recursive validation",
				value, currentPath,
			),
		)}
	}

	msg := v.newMessage()
	if msg == nil {
		return diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
			currentPath,
			"missing recursive message template",
			internalErrorClarification+"validator has no message template to validate value.",
		)}
	}

	unknowns, diags := conversion.MessageFromDynamic(ctx, dynamic, msg, v.nameMap)
	if diags.HasError() {
		return diags
	}

	warnings := map[string]string{}
	checkMessageNIDs(msg.ProtoReflect(), "", v.nidPaths, warnings)
	paths := make([]string, 0, len(warnings))
	for relPath := range warnings {
		paths = append(paths, relPath)
	}
	sort.Strings(paths)
	for _, relPath := range paths {
		if matchesUnknownPath(unknowns, relPath) {
			continue
		}
		diags.AddAttributeWarning(
			currentPath,
			"invalid Nebius ID",
			fmt.Sprintf("Invalid Nebius ID at %s: %s", relPath, warnings[relPath]),
		)
	}
	return diags
}

func checkMessageNIDs(
	msg protoreflect.Message,
	currentPath string,
	nidPaths []*checknid.SubfieldPath,
	warnings map[string]string,
) {
	msg.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldPath := joinNIDPath(currentPath, string(fd.Name()))
		childPaths, contextualNID := advanceNIDPaths(nidPaths, fd)
		if hasNIDResourceSetting(getNIDSettings(fd)) || contextualNID {
			validateNIDField(fd, value, fieldPath, warnings)
		}

		childPaths = append(childPaths, getNIDSubfieldPaths(fd)...)
		if isResourceMetadataField(fd) {
			childPaths = append(childPaths, compileNIDPaths([]string{"parent_id"})...)
		}

		switch {
		case fd.IsMap() && fd.MapValue().Kind() == protoreflect.MessageKind:
			value.Map().Range(func(key protoreflect.MapKey, item protoreflect.Value) bool {
				checkMessageNIDs(item.Message(), mapNIDPath(fieldPath, key.String()), childPaths, warnings)
				return true
			})
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := value.List()
			for i := range list.Len() {
				checkMessageNIDs(list.Get(i).Message(), indexNIDPath(fieldPath, i), childPaths, warnings)
			}
		case fd.Kind() == protoreflect.MessageKind && msg.Has(fd):
			checkMessageNIDs(value.Message(), fieldPath, childPaths, warnings)
		}
		return true
	})
}

func validateNIDField(
	fd protoreflect.FieldDescriptor,
	value protoreflect.Value,
	fieldPath string,
	warnings map[string]string,
) {
	switch {
	case fd.Kind() == protoreflect.StringKind && !fd.IsList():
		if warning := validateNIDFormat(value.String()); warning != "" {
			warnings[fieldPath] = warning
		}
	case fd.IsList() && fd.Kind() == protoreflect.StringKind:
		list := value.List()
		for i := range list.Len() {
			if warning := validateNIDFormat(list.Get(i).String()); warning != "" {
				warnings[indexNIDPath(fieldPath, i)] = warning
			}
		}
	case fd.IsMap() && fd.MapValue().Kind() == protoreflect.StringKind:
		value.Map().Range(func(key protoreflect.MapKey, item protoreflect.Value) bool {
			if warning := validateNIDFormat(item.String()); warning != "" {
				warnings[mapNIDPath(fieldPath, key.String())] = warning
			}
			return true
		})
	}
}

func validateNIDFormat(value string) string {
	if value == "" {
		return ""
	}
	if _, err := nid.Parse(value); err != nil {
		return fmt.Sprintf("value %q is not a valid Nebius ID: %v", value, err)
	}
	return ""
}

func advanceNIDPaths(
	paths []*checknid.SubfieldPath,
	fd protoreflect.FieldDescriptor,
) ([]*checknid.SubfieldPath, bool) {
	next := make([]*checknid.SubfieldPath, 0, len(paths))
	matched := false
	for _, subPath := range paths {
		remaining, ok := subPath.MatchField(fd)
		if !ok || remaining == nil {
			continue
		}
		if remaining.IsEmpty() {
			matched = true
			continue
		}
		next = append(next, remaining)
	}
	return next, matched
}

func getNIDSubfieldPaths(fd protoreflect.FieldDescriptor) []*checknid.SubfieldPath {
	opts, ok := fd.Options().(*descriptorpb.FieldOptions)
	if !ok || !proto.HasExtension(opts, nebius.E_SubfieldSettings) {
		return nil
	}

	settings, ok := proto.GetExtension(opts, nebius.E_SubfieldSettings).([]*nebius.SubfieldSettings)
	if !ok {
		return nil
	}
	paths := make([]string, 0, len(settings))
	for _, setting := range settings {
		if setting == nil {
			continue
		}
		if hasNIDResourceSetting(setting.GetNid()) {
			paths = append(paths, setting.GetFieldPath())
		}
	}
	return compileNIDPaths(paths)
}

func getNIDSettings(fd protoreflect.FieldDescriptor) *nebius.NIDFieldSettings {
	opts, ok := fd.Options().(*descriptorpb.FieldOptions)
	if !ok || !proto.HasExtension(opts, nebius.E_Nid) {
		return nil
	}
	settings, _ := proto.GetExtension(opts, nebius.E_Nid).(*nebius.NIDFieldSettings)
	return settings
}

func hasNIDResourceSetting(settings *nebius.NIDFieldSettings) bool {
	if settings == nil {
		return false
	}
	hasResource := settings.Resource != nil
	return hasResource
}

func isResourceMetadataField(fd protoreflect.FieldDescriptor) bool {
	return fd != nil && !fd.IsList() && !fd.IsMap() &&
		fd.Name() == "metadata" && fd.Kind() == protoreflect.MessageKind &&
		fd.Message().FullName() == "nebius.common.v1.ResourceMetadata"
}

func compileNIDPaths(paths []string) []*checknid.SubfieldPath {
	ret := make([]*checknid.SubfieldPath, 0, len(paths))
	for _, fieldPath := range paths {
		parsed, err := checknid.ParseSubfieldPath(fieldPath)
		if err == nil {
			ret = append(ret, parsed)
		}
	}
	return ret
}

func appendNIDPath(base, child string) string {
	base = strings.TrimSpace(base)
	child = strings.TrimSpace(child)
	if base == "" {
		return child
	}
	if child == "" {
		return base
	}
	return base + "." + child
}

func joinNIDPath(base, child string) string {
	if base == "" {
		return child
	}
	return base + "." + child
}

func indexNIDPath(base string, index int) string {
	return fmt.Sprintf("%s[%d]", base, index)
}

func mapNIDPath(base, key string) string {
	return fmt.Sprintf("%s[%q]", base, key)
}

func matchesUnknownPath(unknowns *mask.Mask, relPath string) bool {
	if unknowns == nil {
		return false
	}
	if unknowns.IsEmpty() {
		return true
	}
	fieldPath, err := parseUnknownPath(relPath)
	return err == nil && fieldPath.MatchesResetMask(unknowns)
}

func parseUnknownPath(value string) (mask.FieldPath, error) {
	if value == "" {
		return mask.FieldPath{}, nil
	}

	ret := make(mask.FieldPath, 0, strings.Count(value, ".")+1)
	for pos := 0; pos < len(value); {
		for pos < len(value) && value[pos] == '.' {
			pos++
		}
		if pos >= len(value) {
			break
		}
		if value[pos] == '[' {
			key, nextPos, err := parseUnknownSubscript(value, pos)
			if err != nil {
				return nil, err
			}
			ret = append(ret, mask.FieldKey(key))
			pos = nextPos
			continue
		}
		start := pos
		for pos < len(value) && value[pos] != '.' && value[pos] != '[' {
			pos++
		}
		if start == pos {
			return nil, strconv.ErrSyntax
		}
		ret = append(ret, mask.FieldKey(value[start:pos]))
	}
	return ret, nil
}

func parseUnknownSubscript(value string, pos int) (string, int, error) {
	if pos >= len(value) || value[pos] != '[' || pos+1 >= len(value) {
		return "", 0, strconv.ErrSyntax
	}
	if value[pos+1] == '"' {
		endQuote := pos + 2
		for endQuote < len(value) {
			if value[endQuote] == '\\' {
				endQuote += 2
				continue
			}
			if value[endQuote] == '"' {
				break
			}
			endQuote++
		}
		if endQuote >= len(value) || endQuote+1 >= len(value) || value[endQuote+1] != ']' {
			return "", 0, strconv.ErrSyntax
		}
		key, err := strconv.Unquote(value[pos+1 : endQuote+1])
		return key, endQuote + 2, err
	}

	end := strings.IndexByte(value[pos+1:], ']')
	if end == -1 {
		return "", 0, strconv.ErrSyntax
	}
	end += pos + 1
	return value[pos+1 : end], end + 1, nil
}
