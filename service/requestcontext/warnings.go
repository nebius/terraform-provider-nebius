package requestcontext

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/osteele/liquid"

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

	var unsupportedToolVersionWarning *commonpb.Warning

	for _, warning := range warnings.GetWarnings() {
		if warning == nil {
			continue
		}
		// Schema validators already report malformed Nebius IDs. Resource type
		// mismatch warnings have a separate code and must remain visible.
		if warning.GetCode() == commonpb.Warning_CODE_INVALID_NEBIUS_ID_FORMAT_REQUEST {
			continue
		}
		if warning.GetCode() == commonpb.Warning_CODE_DEPRECATED_TOOL_VERSION {
			unsupportedToolVersionWarning = warning
			continue
		}

		summary, details := getSummaryAndDetails(warning, engine, bindings)

		attrPath, err := FieldPathToTFPath(
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

	if unsupportedToolVersionWarning != nil {
		summary, details := getSummaryAndDetails(unsupportedToolVersionWarning, engine, bindings)
		var newDiags diag.Diagnostics
		for _, d := range diags {
			// we search in details because gateway returns warning and an error with a summary of the warnings as an error message.
			// terraform add an SeverityError diagnostic with a custom summary and error message as details.
			if d.Severity() == diag.SeverityError && summary != "" && strings.Contains(d.Detail(), summary) {
				newDiags = append(newDiags, diag.NewErrorDiagnostic(summary, details+"\n\n"+d.Detail()))
			} else {
				newDiags = append(newDiags, d)
			}
		}
		diags = newDiags
	}
	return diags
}

func getSummaryAndDetails(warning *commonpb.Warning, engine *liquid.Engine, bindings liquid.Bindings) (string, string) {
	summary := applyWarningTemplate(engine, bindings, warning.GetSummary(), warning.GetSummaryFallback())
	details := applyWarningTemplate(engine, bindings, warning.GetDetails(), warning.GetDetailsFallback())
	return summary, details
}

func fieldName(
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) func(string) (string, error) {
	return func(fieldPath string) (string, error) {
		attrPath, err := FieldPathToTFPath(
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
