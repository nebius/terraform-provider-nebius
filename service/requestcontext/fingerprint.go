package requestcontext

import (
	"strings"
)

type RequestFingerprint struct {
	MainRequestID   string
	MainTraceID     string
	MainTraceUI     string
	LatestRequestID string
	LatestTraceID   string
	LatestTraceUI   string
}

func (f *RequestFingerprint) ContextString() string {
	if f.MainRequestID == "" {
		return ""
	}
	b := strings.Builder{}
	b.WriteString(
		"The following information may be helpful while debugging the issue:\n",
	)
	b.WriteString("Request ID: ")
	b.WriteString(f.MainRequestID)
	if f.MainTraceID != "" {
		b.WriteString("\nTrace ID: ")
		b.WriteString(f.MainTraceID)
	}
	if f.MainTraceUI != "" {
		b.WriteString("\nTrace URL: ")
		b.WriteString(f.MainTraceUI)
	}

	if f.LatestRequestID != "" {
		b.WriteString("\n\nLatest Request ID: ")
		b.WriteString(f.LatestRequestID)
		if f.LatestTraceID != "" {
			b.WriteString("\nLatest Trace ID: ")
			b.WriteString(f.LatestTraceID)
		}
		if f.LatestTraceUI != "" {
			b.WriteString("\nLatest Trace URL: ")
			b.WriteString(f.LatestTraceUI)
		}
	}
	return b.String()
}
