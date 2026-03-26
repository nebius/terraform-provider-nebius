package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func UnwrapDynamic(ctx context.Context, val attr.Value) (attr.Value, bool, diag.Diagnostics) {
	wasDynamic := false
	for {
		dValuable, ok := val.(basetypes.DynamicValuable)
		if ok {
			if dValuable.IsNull() || dValuable.IsUnknown() {
				return val, true, nil
			}
			wasDynamic = true
			var d diag.Diagnostics
			dval, d := dValuable.ToDynamicValue(ctx)
			if d.HasError() {
				return dValuable, true, d
			}
			if dval.IsNull() || dval.IsUnknown() {
				return dval, true, nil
			}
			v := dval.UnderlyingValue()
			if v == nil {
				return dval, true, nil
			}
			val = v
			continue
		}
		break
	}
	return val, wasDynamic, nil
}
