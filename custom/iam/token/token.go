package token

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/timestamp"
	"github.com/nebius/terraform-provider-nebius/provider"
	"github.com/nebius/terraform-provider-nebius/service"
)

type iamTokenResource struct {
	provider provider.Provider
}

// Metadata implements ephemeral.EphemeralResource.
func (i *iamTokenResource) Metadata(
	ctx context.Context,
	req ephemeral.MetadataRequest,
	resp *ephemeral.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_iam_token"
}

// Open implements ephemeral.EphemeralResource.
func (i *iamTokenResource) Open(
	ctx context.Context,
	req ephemeral.OpenRequest,
	resp *ephemeral.OpenResponse,
) {
	tok, err := i.provider.SDK().BearerToken(ctx)
	if err != nil {
		resp.Diagnostics = service.ErrorToDiag(
			resp.Diagnostics, err, "failed to get IAM token",
		)
		return
	}
	tokAttr := types.StringValue(tok.Token)
	expAttr := timestamp.NewTimeStampTimeValue(tok.ExpiresAt)

	resp.Diagnostics.Append(resp.Result.SetAttribute(
		ctx, path.Root("token"), tokAttr,
	)...)
	resp.Diagnostics.Append(resp.Result.SetAttribute(
		ctx, path.Root("expires_at"), expAttr,
	)...)
}

// Schema implements ephemeral.EphemeralResource.
func (i *iamTokenResource) Schema(
	ctx context.Context,
	req ephemeral.SchemaRequest,
	resp *ephemeral.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				Description:         "The IAM access token.",
				MarkdownDescription: "The IAM access token.",
			},
			"expires_at": schema.StringAttribute{
				CustomType:          &timestamp.TimeStampType{},
				Computed:            true,
				Description:         "The expiration time of the token.",
				MarkdownDescription: "The expiration time of the token.",
			},
		},
		Description:         "IAM token resource.",
		MarkdownDescription: "IAM token resource.",
	}
}

var _ ephemeral.EphemeralResource = (*iamTokenResource)(nil)

func NewIAMTokenFactory(
	provider provider.Provider,
) func() ephemeral.EphemeralResource {
	return func() ephemeral.EphemeralResource {
		return &iamTokenResource{
			provider: provider,
		}
	}
}
