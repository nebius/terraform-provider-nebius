package requestcontext

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	commonpb "github.com/nebius/gosdk/proto/nebius/common/v1"
)

const (
	requestIDKey    = "X-Request-ID"
	traceIDKey      = "X-Trace-ID"
	traceUIKey      = "X-Trace-UI"
	warningsTrailer = "X-Nebius-Warnings-Bin"
)

type Context struct {
	MainHeaders    *metadata.MD
	LatestHeaders  *metadata.MD
	MainTrailers   *metadata.MD
	LatestTrailers *metadata.MD
}

func (c *Context) MainRequestOptions() []grpc.CallOption {
	if c.MainHeaders == nil {
		c.MainHeaders = &metadata.MD{}
	}
	if c.MainTrailers == nil {
		c.MainTrailers = &metadata.MD{}
	}
	return []grpc.CallOption{grpc.Header(c.MainHeaders), grpc.Trailer(c.MainTrailers)}
}

func (c *Context) SubsequentRequestOptions() []grpc.CallOption {
	if c.LatestHeaders == nil {
		c.LatestHeaders = &metadata.MD{}
	}
	if c.LatestTrailers == nil {
		c.LatestTrailers = &metadata.MD{}
	}
	return []grpc.CallOption{grpc.Header(c.LatestHeaders), grpc.Trailer(c.LatestTrailers)}
}

func (c *Context) RequestWarnings() *commonpb.Warnings {
	warnings := &commonpb.Warnings{}
	if c.MainTrailers != nil {
		collectWarningsFromMD(warnings, c.MainTrailers)
	}

	if c.LatestTrailers != nil {
		collectWarningsFromMD(warnings, c.LatestTrailers)
	}

	if len(warnings.GetWarnings()) == 0 {
		return nil
	}
	return warnings
}

func collectWarningsFromMD(dst *commonpb.Warnings, trailers *metadata.MD) {
	for _, rawWarnings := range trailers.Get(warningsTrailer) {
		extraWarnings := &commonpb.Warnings{}
		if err := proto.Unmarshal([]byte(rawWarnings), extraWarnings); err != nil {
			tflog.Warn(context.Background(), "failed to unmarshal warnings", map[string]any{"error": err})
			continue
		}
		dst.Warnings = append(dst.Warnings, extraWarnings.GetWarnings()...)
	}
}

func MergeContexts(primary, secondary *Context) *Context {
	if primary == nil {
		return secondary
	}
	if secondary == nil {
		return primary
	}
	return &Context{
		MainHeaders:    mergeMD(primary.MainHeaders, secondary.MainHeaders),
		LatestHeaders:  mergeMD(primary.LatestHeaders, secondary.LatestHeaders),
		MainTrailers:   mergeMD(primary.MainTrailers, secondary.MainTrailers),
		LatestTrailers: mergeMD(primary.LatestTrailers, secondary.LatestTrailers),
	}
}

func mergeMD(first, second *metadata.MD) *metadata.MD {
	if first == nil {
		if second == nil {
			return nil
		}
		md := metadata.Join(*second)
		return &md
	}
	if second == nil {
		md := metadata.Join(*first)
		return &md
	}
	md := metadata.Join(*first, *second)
	return &md
}

func (c *Context) RequestFingerprint() *RequestFingerprint {
	ret := RequestFingerprint{}
	if c == nil {
		return &ret
	}

	if c.MainHeaders != nil {
		mainRequestIDs := c.MainHeaders.Get(requestIDKey)
		mainTraceIDs := c.MainHeaders.Get(traceIDKey)
		mainTraceUIs := c.MainHeaders.Get(traceUIKey)
		if len(mainRequestIDs) > 0 {
			ret.MainRequestID = mainRequestIDs[0]
		}
		if len(mainTraceIDs) > 0 {
			ret.MainTraceID = mainTraceIDs[0]
		}
		if len(mainTraceUIs) > 0 {
			ret.MainTraceUI = mainTraceUIs[0]
		}
	}

	if c.LatestHeaders != nil {
		latestRequestIDs := c.LatestHeaders.Get(requestIDKey)
		latestTraceIDs := c.LatestHeaders.Get(traceIDKey)
		latestTraceUIs := c.LatestHeaders.Get(traceUIKey)
		if len(latestRequestIDs) > 0 {
			ret.LatestRequestID = latestRequestIDs[0]
		}
		if len(latestTraceIDs) > 0 {
			ret.LatestTraceID = latestTraceIDs[0]
		}
		if len(latestTraceUIs) > 0 {
			ret.LatestTraceUI = latestTraceUIs[0]
		}
	}

	return &ret
}

func (c *Context) wrapDiagnostic(d diag.Diagnostic) diag.Diagnostic {
	ret := wrapDiagnostic{
		fingerprint: c.RequestFingerprint(),
		wrapped:     d,
	}
	if dp, ok := d.(diag.DiagnosticWithPath); ok {
		return wrapDiagnosticWithPath{
			wrapDiagnostic: ret,
			wrappedPath:    dp,
		}
	}
	return ret
}

func (c *Context) WrapDiagnostics(
	diags diag.Diagnostics,
	schemaType attr.Type,
	fieldNameMap map[string]map[string]string,
) diag.Diagnostics {
	diags = processWarnings(c.RequestWarnings(), diags, schemaType, fieldNameMap)

	fp := c.RequestFingerprint()
	if fp == nil || fp.MainRequestID == "" || len(diags) == 0 {
		return diags
	}
	if !diags.HasError() {
		if len(diags) == 1 {
			diags[0] = c.wrapDiagnostic(diags[0])
			return diags
		}
		diags.Append(&additionalDiagnostic{
			fingerprint: fp,
			severity:    diag.SeverityWarning,
		})
		return diags
	} else if len(diags.Errors()) == 1 {
		for i, d := range diags {
			if d.Severity() == diag.SeverityError {
				diags[i] = c.wrapDiagnostic(d)
				return diags
			}
		}
	}
	diags.Append(&additionalDiagnostic{
		fingerprint: fp,
		severity:    diag.SeverityError,
	})
	return diags
}
