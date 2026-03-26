package conversion

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

type DiagnosticsError struct {
	Diagnostics diag.Diagnostics
}

func (e *DiagnosticsError) Error() string {
	b := strings.Builder{}
	b.WriteString("diagnostics:")
	for _, d := range e.Diagnostics {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("[%7s]", d.Severity()))
		b.WriteString(": ")
		b.WriteString(d.Summary())
		b.WriteString(" - ")
		b.WriteString(d.Detail())
	}
	return b.String()
}

func NewDiagnosticsError(d diag.Diagnostics) error {
	return &DiagnosticsError{
		Diagnostics: d,
	}
}

func DiagnosticsFromErrString(summary, description string) diag.Diagnostics {
	d := diag.Diagnostics{}
	d.AddError(summary, description)
	return d
}

func DiagnosticsFromAttributeErrString(at path.Path, summary, description string) diag.Diagnostics {
	d := diag.Diagnostics{}
	d.AddAttributeError(at, summary, description)
	return d
}
