package validators

import (
	"context"
	"math"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type uint64Validator struct{}

func Uint64Validator() validator.Number {
	return &uint64Validator{}
}

func (v *uint64Validator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (*uint64Validator) MarkdownDescription(_ context.Context) string {
	return "requires to fit inside `Uint64` bounds"
}

func (v *uint64Validator) ValidateNumber(ctx context.Context, req validator.NumberRequest, resp *validator.NumberResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueBigFloat()

	if val.Cmp(big.NewFloat(0)) < 0 || val.Cmp(big.NewFloat(0).SetPrec(0).SetUint64(math.MaxUint64)) > 0 {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			v.Description(ctx),
			req.ConfigValue.String(),
		))
	}
}

func MapUint64Validator() validator.Map {
	return mapvalidator.ValueNumbersAre(Uint64Validator())
}

func ListUint64Validator() validator.List {
	return listvalidator.ValueNumbersAre(Uint64Validator())
}
