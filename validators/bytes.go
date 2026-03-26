package validators

import (
	"context"
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type bytesValidator struct{}

func BytesValidator() validator.String {
	return &bytesValidator{}
}

func (v *bytesValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (*bytesValidator) MarkdownDescription(_ context.Context) string {
	return "must be valid base64 string as declared in RFC 4648"
}

func (v *bytesValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	_, err := base64.StdEncoding.DecodeString(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			v.Description(ctx),
			req.ConfigValue.String(),
		))
	}
}

func MapBytesValidator() validator.Map {
	return mapvalidator.ValueStringsAre(BytesValidator())
}

func ListBytesValidator() validator.List {
	return listvalidator.ValueStringsAre(BytesValidator())
}
