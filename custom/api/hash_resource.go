package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/nebius/terraform-provider-nebius/service"
)

type gettable interface {
	Get(ctx context.Context, target interface{}) diag.Diagnostics
}

type settable interface {
	Set(ctx context.Context, target interface{}) diag.Diagnostics
}

type removable interface {
	RemoveResource(ctx context.Context)
}

type hashResource struct {
	versionedEphemerals map[string]attr.Value
}

type hashResourceConfig struct {
	Name types.String `tfsdk:"name"`
	Hash types.String `tfsdk:"hash"`
}

func generateSalt() (string, error) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	for i := range salt {
		salt[i] &= 0x7f
	}
	return string(salt), nil
}

func getOrGenerateSalt(
	ctx context.Context,
	storage service.TfKeyValueStorage,
) (types.String, diag.Diagnostics) {
	var saltStr string
	found, diag := service.GetObject(ctx, storage, "salt", &saltStr)
	if diag.HasError() {
		return types.String{}, diag
	}
	if !found || saltStr == "" {
		salt, err := generateSalt()
		if err != nil {
			diag.AddError(
				"Failed to generate salt",
				err.Error(),
			)
			return types.String{}, diag
		}
		saltStr = salt
		diag = service.SetObject(ctx, storage, "salt", saltStr)
		if diag.HasError() {
			return types.String{}, diag
		}
	}
	saltString := types.StringValue(saltStr)
	return saltString, nil
}

func stringOfValue(val tftypes.Value, path path.Path) (string, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if val.IsNull() {
		return "null", diags
	}
	if !val.IsKnown() {
		diags.AddError(
			"unknown value",
			fmt.Sprintf(
				"versioned ephemeral value at path %s is unknown",
				path,
			),
		)
		return "", diags
	}
	typ := val.Type()
	switch {
	case typ.Is(tftypes.String):
		var s string
		err := val.As(&s)
		if err != nil {
			diags.AddError(
				"failed to convert value to string",
				fmt.Sprintf(
					"failed to convert value at path %s to string: %s",
					path, err,
				),
			)
			return "", diags
		}
		return fmt.Sprintf("%q", s), diags
	case typ.Is(tftypes.Bool):
		var b bool
		err := val.As(&b)
		if err != nil {
			diags.AddError(
				"failed to convert value to bool",
				fmt.Sprintf(
					"failed to convert value at path %s to bool: %s", path, err,
				),
			)
			return "", diags
		}
		return fmt.Sprintf("%t", b), diags
	case typ.Is(tftypes.Number):
		var n big.Float
		err := val.As(&n)
		if err != nil {
			diags.AddError(
				"failed to convert value to number",
				fmt.Sprintf(
					"failed to convert value at path %s to number: %s", path, err,
				),
			)
			return "", diags
		}
		return n.Text('g', -1), diags
	case typ.Is(tftypes.List{}), typ.Is(tftypes.Set{}), typ.Is(tftypes.Tuple{}):
		var l []tftypes.Value
		err := val.As(&l)
		if err != nil {
			diags.AddError(
				"failed to convert value to list",
				fmt.Sprintf("failed to convert value at path %s to list: %s", path, err),
			)
			return "", diags
		}
		items := make([]string, len(l))
		for i, item := range l {
			itemStr, itemDiags := stringOfValue(item, path.AtListIndex(i))
			diags.Append(itemDiags...)
			items[i] = itemStr
		}
		name := "list"
		if typ.Is(tftypes.Set{}) {
			name = "set"
			sort.Strings(items)
		} else if typ.Is(tftypes.Tuple{}) {
			name = "tuple"
		}
		return fmt.Sprintf("(%s)%v", name, items), diags
	case typ.Is(tftypes.Map{}) || typ.Is(tftypes.Object{}):
		var m map[string]tftypes.Value
		err := val.As(&m)
		if err != nil {
			diags.AddError(
				"failed to convert value to map",
				fmt.Sprintf("failed to convert value at path %s to map: %s", path, err),
			)
			return "", diags
		}
		keys := slices.Sorted(maps.Keys(m))
		items := make([]string, len(keys))
		for i, k := range keys {
			itemStr, itemDiags := stringOfValue(m[k], path.AtMapKey(k))
			diags.Append(itemDiags...)
			items[i] = fmt.Sprintf("%q: %s", k, itemStr)
		}
		name := "map"
		if typ.Is(tftypes.Object{}) {
			name = "object"
		}
		return fmt.Sprintf("(%s)%v", name, items), diags
	default:
		diags.AddError(
			"unsupported value type",
			fmt.Sprintf("unsupported value type at path %s: %s", path, typ.String()),
		)
		return "", diags
	}
}

func (s *hashResource) hash(
	ctx context.Context,
	name types.String,
	salt types.String,
	removeResource removable,
) (types.String, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if name.IsNull() || name.IsUnknown() {
		diags.AddError(
			"name is unknown or null",
			"name is unknown or null",
		)
		return types.String{}, diags
	}
	if salt.IsNull() || salt.IsUnknown() {
		diags.AddError(
			"salt is unknown or null",
			"salt is unknown or null",
		)
		return types.String{}, diags
	}
	val, ok := s.versionedEphemerals[name.ValueString()]
	if !ok {
		if removeResource != nil {
			removeResource.RemoveResource(ctx)
			return types.StringValue(""), diags
		}
		diags.AddError(
			"versioned ephemeral value not found",
			fmt.Sprintf(
				"versioned ephemeral value %q not found, possible values: %v",
				name.ValueString(),
				slices.Sorted(maps.Keys(s.versionedEphemerals)),
			),
		)
		return types.String{}, diags
	}
	tfVal, err := val.ToTerraformValue(ctx)
	if err != nil {
		diags.AddError(
			"failed to convert versioned ephemeral value to terraform value",
			fmt.Sprintf(
				"failed to convert versioned ephemeral value %q to terraform value: %s",
				name.ValueString(), err,
			),
		)
		return types.String{}, diags
	}
	valStr, valDiags := stringOfValue(tfVal, path.Root(name.ValueString()))
	diags.Append(valDiags...)
	if diags.HasError() {
		return types.String{}, diags
	}
	hash := sha256.Sum256([]byte(salt.ValueString() + valStr))
	return types.StringValue(hex.EncodeToString(hash[:])), diags
}

func (s *hashResource) hashAndSet(
	ctx context.Context,
	inputState gettable,
	outputState settable,
	private service.TfKeyValueStorage,
	removeResource removable,
) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var config hashResourceConfig
	diags.Append(inputState.Get(ctx, &config)...)
	if diags.HasError() {
		return diags
	}
	salt, diag := getOrGenerateSalt(ctx, private)
	diags.Append(diag...)
	if diags.HasError() {
		return diags
	}
	hash, diag := s.hash(ctx, config.Name, salt, removeResource)
	diags.Append(diag...)
	if diags.HasError() {
		return diags
	}
	config.Hash = hash
	diags.Append(outputState.Set(ctx, config)...)
	return diags
}

func (s *hashResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	diags := s.hashAndSet(ctx, req.Plan, &resp.State, resp.Private, nil)
	resp.Diagnostics.Append(diags...)
}

func (s *hashResource) Delete(
	_ context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	// do nothing
}

func (s *hashResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_hash"
}

func (s *hashResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	diags := s.hashAndSet(ctx, req.State, &resp.State, resp.Private, &resp.State)
	resp.Diagnostics.Append(diags...)
}

func (s *hashResource) Schema(
	_ context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "A salted SHA256 hash of a versioned ephemeral value that was passed to the provider",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the versioned ephemeral value",
				Required:    true,
			},
			"hash": schema.StringAttribute{
				Description: "The hash of the versioned ephemeral value",
				Computed:    true,
			},
		},
	}
}

func (s *hashResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	diags := s.hashAndSet(ctx, req.Plan, &resp.State, resp.Private, nil)
	resp.Diagnostics.Append(diags...)
}

func NewHashResource(
	versionedEphemerals map[string]attr.Value,
) resource.Resource {
	return &hashResource{
		versionedEphemerals: versionedEphemerals,
	}
}
