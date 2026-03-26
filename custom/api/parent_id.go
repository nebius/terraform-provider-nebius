package api

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

type parentID struct {
	parentID string
}

// Metadata implements datasource.DataSource.
func (p *parentID) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parent_id"
}

// Read implements datasource.DataSource.
func (p *parentID) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("parent_id"), p.parentID,
	)...)
}

// Schema implements datasource.DataSource.
func (p *parentID) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"parent_id": schema.StringAttribute{
				Computed:    true,
				Description: "The default parent ID from the configuration.",
			},
		},
		Description: "A data source that provides the default parent ID from the configuration.",
	}
}

func NewParentID(parentIDstr string) datasource.DataSource {
	return &parentID{
		parentID: parentIDstr,
	}
}
