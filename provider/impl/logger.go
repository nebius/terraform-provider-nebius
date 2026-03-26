package provider

import (
	"context"
	"log/slog"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type slogHandler struct {
	attrs []slog.Attr
}

var _ slog.Handler = (*slogHandler)(nil)

func (*slogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *slogHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := make(map[string]any, len(h.attrs)+record.NumAttrs())
	addAttrsToMap(h.attrs, fields)

	record.Attrs(func(attr slog.Attr) bool {
		addAttrToMap(attr, fields)
		return true
	})

	switch record.Level {
	case slog.LevelDebug:
		tflog.Debug(ctx, record.Message, fields)
	case slog.LevelInfo:
		tflog.Info(ctx, record.Message, fields)
	case slog.LevelWarn:
		tflog.Warn(ctx, record.Message, fields)
	case slog.LevelError:
		tflog.Error(ctx, record.Message, fields)
	default:
		tflog.Info(ctx, record.Message, fields)
	}
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogHandler{
		attrs: append(h.attrs, attrs...),
	}
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &slogHandlerWithGroups{
		next:   h,
		groups: []string{name},
	}
}

type slogHandlerWithGroups struct {
	next   slog.Handler
	groups []string
}

var _ slog.Handler = (*slogHandlerWithGroups)(nil)

func (*slogHandlerWithGroups) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *slogHandlerWithGroups) Handle(ctx context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	attr := slog.Attr{
		Key:   h.groups[0],
		Value: slog.GroupValue(attrs...),
	}
	for _, group := range h.groups[1:] {
		attr = slog.Attr{
			Key:   group,
			Value: slog.GroupValue(attr),
		}
	}

	rec := slog.Record{
		Time:    record.Time,
		Message: record.Message,
		Level:   record.Level,
		PC:      record.PC,
	}
	rec.AddAttrs(attr)
	return h.next.Handle(ctx, rec)
}

func (h *slogHandlerWithGroups) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogHandlerWithAttrs{
		next:  h,
		attrs: attrs,
	}
}

func (h *slogHandlerWithGroups) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &slogHandlerWithGroups{
		next:   h.next,
		groups: append([]string{name}, h.groups...),
	}
}

type slogHandlerWithAttrs struct {
	next  slog.Handler
	attrs []slog.Attr
}

var _ slog.Handler = (*slogHandlerWithAttrs)(nil)

func (*slogHandlerWithAttrs) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *slogHandlerWithAttrs) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(h.attrs...)
	return h.next.Handle(ctx, record)
}

func (h *slogHandlerWithAttrs) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogHandlerWithAttrs{
		next:  h.next,
		attrs: append(h.attrs, attrs...),
	}
}

func (h *slogHandlerWithAttrs) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &slogHandlerWithGroups{
		next:   h,
		groups: []string{name},
	}
}

func addAttrsToMap(attrs []slog.Attr, fields map[string]any) {
	for _, a := range attrs {
		addAttrToMap(a, fields)
	}
}

func addAttrToMap(attr slog.Attr, fields map[string]any) {
	if attr.Equal(slog.Attr{}) {
		return
	}

	val := attr.Value.Resolve()

	if val.Kind() == slog.KindGroup {
		attrs := val.Group()
		if len(attrs) == 0 {
			return
		}

		if attr.Key == "" {
			addAttrsToMap(attrs, fields)
			return
		}

		group := make(map[string]any, len(attrs))
		addAttrsToMap(attrs, group)
		fields[attr.Key] = group

		return
	}

	fields[attr.Key] = val.Any()
}
