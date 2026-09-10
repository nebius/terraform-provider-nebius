package requestcontext

import (
	"errors"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/nebius/gosdk/serviceerror"
)

var _ diag.Diagnostic = (*formattableDiagnostic)(nil)

type formattableDiagnostic struct {
	summary           string
	message           string
	fieldPath         string
	relatedFieldPaths []string
}

func (d *formattableDiagnostic) Severity() diag.Severity {
	return diag.SeverityError
}

func (d *formattableDiagnostic) Summary() string {
	return d.summary
}

func (d *formattableDiagnostic) Detail() string {
	return d.message
}

func (d *formattableDiagnostic) Equal(other diag.Diagnostic) bool {
	o, ok := other.(*formattableDiagnostic)
	return ok && d.summary == o.summary &&
		d.message == o.message &&
		d.fieldPath == o.fieldPath &&
		slices.Equal(d.relatedFieldPaths, o.relatedFieldPaths)
}

// ExtractFormattableDiagnostics extracts structured BadRequest diagnostics and
// returns any unhandled details for the caller to report normally.
func ExtractFormattableDiagnostics(err error, summary string) (diag.Diagnostics, error) {
	var serviceErr *serviceerror.Error
	if !errors.As(err, &serviceErr) {
		return nil, err
	}

	var diagnostics diag.Diagnostics
	remainingDetails := make([]serviceerror.Detail, 0, len(serviceErr.Details))
	for _, detail := range serviceErr.Details {
		previousLen := len(diagnostics)
		switch typed := detail.(type) {
		case *serviceerror.BadRequest:
			for _, violation := range typed.Violations() {
				if violation == nil {
					continue
				}
				diagnostics.Append(&formattableDiagnostic{
					summary:           summary,
					message:           violation.GetMessage(),
					fieldPath:         violation.GetField(),
					relatedFieldPaths: slices.Clone(violation.GetRelatedFields()),
				})
			}
		}
		if len(diagnostics) == previousLen {
			remainingDetails = append(remainingDetails, detail)
		}
	}
	if len(diagnostics) == 0 {
		return nil, err
	}
	if len(remainingDetails) == 0 {
		return diagnostics, nil
	}
	return diagnostics, &serviceerror.Error{
		Wrapped:   serviceErr.Wrapped,
		Details:   remainingDetails,
		RequestID: serviceErr.RequestID,
	}
}

func resolveFormattableDiagnostics(
	diagnostics diag.Diagnostics,
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) diag.Diagnostics {
	resolved := make(diag.Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		formattable, ok := diagnostic.(*formattableDiagnostic)
		if !ok {
			resolved.Append(diagnostic)
			continue
		}
		details := formatViolationDetails(formattable, schemaType, fieldNameMap)
		attrPath, err := FieldPathToTFPath(
			formattable.fieldPath,
			schemaType,
			fieldNameMap,
		)
		if err != nil {
			resolved.Append(diag.NewErrorDiagnostic(formattable.summary, details))
			continue
		}
		resolved.Append(diag.NewAttributeErrorDiagnostic(
			attrPath,
			formattable.summary,
			details,
		))
	}
	return resolved
}

func formatViolationDetails(
	diagnostic *formattableDiagnostic,
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) string {
	fields := relatedAndPrimaryFields(diagnostic.relatedFieldPaths, diagnostic.fieldPath)
	if len(fields) < 2 {
		return diagnostic.message
	}
	displayFields := make([]string, 0, len(fields))
	for _, fieldPath := range fields {
		attrPath, err := FieldPathToTFPath(fieldPath, schemaType, fieldNameMap)
		if err != nil {
			displayFields = append(displayFields, fieldPath)
			continue
		}
		displayFields = append(displayFields, attrPath.String())
	}
	fieldPrefix := strings.Join(displayFields, ", ")
	if diagnostic.message == "" {
		return fieldPrefix
	}
	return fieldPrefix + ": " + diagnostic.message
}

func relatedAndPrimaryFields(relatedFields []string, field string) []string {
	fields := make([]string, 0, len(relatedFields)+1)
	seen := make(map[string]struct{}, len(relatedFields)+1)
	appendField := func(path string) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		fields = append(fields, path)
	}
	for _, path := range relatedFields {
		appendField(path)
	}
	appendField(field)
	return fields
}
