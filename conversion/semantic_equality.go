package conversion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
)

func SemanticallyEqual(
	ctx context.Context, a, b attr.Value,
) (bool, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if a.Equal(b) {
		return true, diags
	}
	if !a.Type(ctx).Equal(b.Type(ctx)) {
		return false, diags
	}
	switch aTyped := a.(type) {
	case basetypes.BoolValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.BoolValuable); ok {
			return aTyped.BoolSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.Float64ValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.Float64Valuable); ok {
			return aTyped.Float64SemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.Int64ValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.Int64Valuable); ok {
			return aTyped.Int64SemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.ListValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.ListValuable); ok {
			return aTyped.ListSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.MapValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.MapValuable); ok {
			return aTyped.MapSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.NumberValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.NumberValuable); ok {
			return aTyped.NumberSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.ObjectValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.ObjectValuable); ok {
			return aTyped.ObjectSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.SetValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.SetValuable); ok {
			return aTyped.SetSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.StringValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.StringValuable); ok {
			return aTyped.StringSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.DynamicValuableWithSemanticEquals:
		if bTyped, ok := b.(basetypes.DynamicValuable); ok {
			return aTyped.DynamicSemanticEquals(ctx, bTyped)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.ListValuable:
		if bTyped, ok := b.(basetypes.ListValuable); ok {
			aList, inDiag := aTyped.ToListValue(ctx)
			diags.Append(inDiag...)
			bList, inDiag := bTyped.ToListValue(ctx)
			diags.Append(inDiag...)
			if diags.HasError() {
				return false, diags
			}
			aElements := aList.Elements()
			bElements := bList.Elements()
			if len(aElements) != len(bElements) {
				return false, diags
			}
			for i, el := range aElements {
				isEqual, inDiag := SemanticallyEqual(ctx, el, bElements[i])
				diags.Append(inDiag...)
				if !isEqual || diags.HasError() {
					return false, diags
				}
			}
			return true, diags
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.SetValuable:
		if bTyped, ok := b.(basetypes.SetValuable); ok {
			aList, inDiag := aTyped.ToSetValue(ctx)
			diags.Append(inDiag...)
			bList, inDiag := bTyped.ToSetValue(ctx)
			diags.Append(inDiag...)
			if diags.HasError() {
				return false, diags
			}
			aElements := aList.Elements()
			bElements := bList.Elements()
			if len(aElements) != len(bElements) {
				return false, diags
			}
			for _, aEl := range aElements {
				found := false
				for _, bEl := range bElements {
					isEqual, inDiag := SemanticallyEqual(ctx, aEl, bEl)
					diags.Append(inDiag...)
					if diags.HasError() {
						return false, diags
					}
					if isEqual {
						found = true
						break
					}
				}
				if !found {
					return false, diags
				}
			}
			return true, diags
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.MapValuable:
		if bTyped, ok := b.(basetypes.MapValuable); ok {
			aList, inDiag := aTyped.ToMapValue(ctx)
			diags.Append(inDiag...)
			bList, inDiag := bTyped.ToMapValue(ctx)
			diags.Append(inDiag...)
			if diags.HasError() {
				return false, diags
			}
			aElements := aList.Elements()
			bElements := bList.Elements()
			if len(aElements) != len(bElements) {
				return false, diags
			}
			for key, aEl := range aElements {
				if bEl, found := bElements[key]; found {
					isEqual, inDiag := SemanticallyEqual(ctx, aEl, bEl)
					diags.Append(inDiag...)
					if !isEqual || diags.HasError() {
						return false, diags
					}
					continue
				}
				return false, diags
			}
			return true, diags
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.ObjectValuable:
		if bTyped, ok := b.(basetypes.ObjectValuable); ok {
			aList, inDiag := aTyped.ToObjectValue(ctx)
			diags.Append(inDiag...)
			bList, inDiag := bTyped.ToObjectValue(ctx)
			diags.Append(inDiag...)
			if diags.HasError() {
				return false, diags
			}
			aElements := aList.Attributes()
			bElements := bList.Attributes()
			if len(aElements) != len(bElements) {
				return false, diags
			}
			for key, aEl := range aElements {
				if bEl, found := bElements[key]; found {
					isEqual, inDiag := SemanticallyEqual(ctx, aEl, bEl)
					diags.Append(inDiag...)
					if !isEqual || diags.HasError() {
						return false, diags
					}
					continue
				}
				return false, diags
			}
			return true, diags
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	case basetypes.DynamicValuable:
		if bTyped, ok := b.(basetypes.DynamicValuable); ok {
			aUnderlying, _, inDiag := ctypes.UnwrapDynamic(ctx, aTyped)
			diags.Append(inDiag...)
			bUnderlying, _, inDiag := ctypes.UnwrapDynamic(ctx, bTyped)
			diags.Append(inDiag...)
			if diags.HasError() {
				return false, diags
			}
			return SemanticallyEqual(
				ctx,
				aUnderlying,
				bUnderlying,
			)
		}
		diags.AddError(
			"failed to cast second value",
			fmt.Sprintf("both values are checked to be of same type, but the "+
				"second couldn't have been cast to the same interface: "+
				"required inteface %T, received %T", aTyped, b),
		)
		return false, diags
	default:
		return false, diags
	}
}
