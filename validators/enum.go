package validators

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type enumValidator struct {
	names map[string]int32
}

// Description implements validator.String.
func (e *enumValidator) Description(ctx context.Context) string {
	return e.MarkdownDescription(ctx)
}

// MarkdownDescription implements validator.String.
func (e *enumValidator) MarkdownDescription(context.Context) string {
	keys := make([]string, 0, len(e.names))
	for k := range e.names {
		keys = append(keys, k)
	}
	return fmt.Sprintf("value must be one of: %s", strings.Join(keys, ", "))
}

var unknownRegExp = regexp.MustCompile(`^Unknown\[(0|-?[1-9][0-9]*)\]$`)

// ValidateString implements validator.String.
func (e *enumValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	res *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	_, ok := e.names[value]
	if ok || unknownRegExp.MatchString(value) {
		return
	}

	res.Diagnostics.Append(validatordiag.InvalidAttributeValueMatchDiagnostic(
		req.Path,
		e.Description(ctx),
		req.ConfigValue.String(),
	))
}

var _ validator.String = (*enumValidator)(nil)

func EnumValidator(enumValues map[string]int32) validator.String {
	return &enumValidator{names: enumValues}
}

func MapEnumValidator(enumValues map[string]int32) validator.Map {
	return mapvalidator.ValueStringsAre(EnumValidator(enumValues))
}

func ListEnumValidator(enumValues map[string]int32) validator.List {
	return listvalidator.ValueStringsAre(EnumValidator(enumValues))
}
