package types

import "github.com/nebius/gosdk/proto/fieldmask/mask"

func AppendUnknownMask(dst *mask.Mask, prefix mask.FieldPath, src *mask.Mask) *mask.Mask {
	if src == nil {
		return dst
	}
	if src.IsEmpty() {
		return AppendUnknownPath(dst, prefix)
	}
	for key, inner := range src.FieldParts {
		dst = AppendUnknownMask(dst, prefix.Join(key), inner)
	}
	if src.Any != nil {
		dst = AppendUnknownMask(dst, prefix.Join(mask.FieldKey("*")), src.Any)
	}
	return dst
}

func AppendUnknownPath(dst *mask.Mask, path mask.FieldPath) *mask.Mask {
	if len(path) == 0 {
		return mask.New()
	}
	if dst != nil && dst.IsEmpty() {
		return dst
	}
	if dst == nil {
		return path.ToMask()
	}
	_ = dst.Merge(path.ToMask())
	return dst
}
