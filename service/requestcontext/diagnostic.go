package requestcontext

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

var _ diag.Diagnostic = wrapDiagnostic{}
var _ diag.Diagnostic = additionalDiagnostic{}
var _ diag.DiagnosticWithPath = wrapDiagnosticWithPath{}

type wrapDiagnostic struct {
	fingerprint *RequestFingerprint
	wrapped     diag.Diagnostic
}
type wrapDiagnosticWithPath struct {
	wrapDiagnostic
	wrappedPath diag.DiagnosticWithPath
}

// Equal returns true if the other diagnostic is wholly equivalent.
func (d wrapDiagnostic) Equal(other diag.Diagnostic) bool {
	if o, ok := other.(wrapDiagnostic); ok {
		return d.wrapped.Equal(o.wrapped) &&
			((d.fingerprint == nil && o.fingerprint == nil) ||
				(*d.fingerprint == *o.fingerprint))
	}
	if o, ok := other.(wrapDiagnosticWithPath); ok {
		return d.wrapped.Equal(o.wrapped) &&
			((d.fingerprint == nil && o.fingerprint == nil) ||
				(*d.fingerprint == *o.fingerprint))
	}
	return false
}

// Detail returns the diagnostic detail.
func (d wrapDiagnostic) Detail() string {
	if d.fingerprint != nil && d.fingerprint.MainRequestID != "" {
		return d.wrapped.Detail() +
			"\n\nThis issue may be linked to the API request call. " +
			d.fingerprint.ContextString()
	}
	return d.wrapped.Detail()
}

// Severity returns the diagnostic severity.
func (d wrapDiagnostic) Severity() diag.Severity {
	return d.wrapped.Severity()
}

// Summary returns the diagnostic summary.
func (d wrapDiagnostic) Summary() string {
	return d.wrapped.Summary()
}

func (d wrapDiagnosticWithPath) Path() path.Path {
	return d.wrappedPath.Path()
}

type additionalDiagnostic struct {
	fingerprint *RequestFingerprint
	severity    diag.Severity
}

// Equal returns true if the other diagnostic is wholly equivalent.
func (d additionalDiagnostic) Equal(other diag.Diagnostic) bool {
	if o, ok := other.(additionalDiagnostic); ok {
		return *d.fingerprint == *o.fingerprint && d.severity == o.severity
	}
	return false
}

// Detail returns the diagnostic detail.
func (d additionalDiagnostic) Detail() string {
	return "One of the issues above may be linked to the API request call. " +
		d.fingerprint.ContextString()
}

// Severity returns the diagnostic severity.
func (d additionalDiagnostic) Severity() diag.Severity {
	return d.severity
}

// Summary returns the diagnostic summary.
func (d additionalDiagnostic) Summary() string {
	return "Request context"
}
