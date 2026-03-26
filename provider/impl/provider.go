package provider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nebius/gosdk"
	"github.com/nebius/gosdk/auth"
	"github.com/nebius/gosdk/config/paths"
	"github.com/nebius/gosdk/config/reader"
	"github.com/nebius/gosdk/conn"
	"github.com/nebius/gosdk/constants"

	sdkconfig "github.com/nebius/gosdk/config"
	iampb "github.com/nebius/gosdk/proto/nebius/iam/v1"
	iam "github.com/nebius/gosdk/services/nebius/iam/v1"
	ctypes "github.com/nebius/terraform-provider-nebius/conversion/types"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown"
	"github.com/nebius/terraform-provider-nebius/conversion/wellknown/duration"
	"github.com/nebius/terraform-provider-nebius/custom/api"
	"github.com/nebius/terraform-provider-nebius/custom/iam/token"
	"github.com/nebius/terraform-provider-nebius/generated/nebius"
	"github.com/nebius/terraform-provider-nebius/provider/version"
)

const (
	Name            = "nebius"
	Address         = "registry.terraform.io/nebius/" + Name
	defaultClientID = "terraform-provider"

	disableWriteOnlyEnv = "NEBIUS_TERRAFORM_PROVIDER_DISABLE_WRITE_ONLY"
	disableDynamicEnv   = "NEBIUS_TERRAFORM_PROVIDER_DISABLE_DYNAMIC"
)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &internalProvider{}
	}
}

type internalProvider struct {
	sdk                 *gosdk.SDK
	versionedEphemerals map[string]attr.Value
	parentID            string
}

var _ provider.ProviderWithEphemeralResources = (*internalProvider)(nil)

type saConfig struct {
	CredentialsFile    types.String `tfsdk:"credentials_file"`
	CredentialsFileEnv types.String `tfsdk:"credentials_file_env"`
	PrivateKey         types.String `tfsdk:"private_key"`
	PrivateKeyFile     types.String `tfsdk:"private_key_file"`
	PrivateKeyFileEnv  types.String `tfsdk:"private_key_file_env"`
	PublicKeyID        types.String `tfsdk:"public_key_id"`
	PublicKeyIDEnv     types.String `tfsdk:"public_key_id_env"`
	AccountID          types.String `tfsdk:"account_id"`
	AccountIDEnv       types.String `tfsdk:"account_id_env"`
}

type profileConfig struct {
	Name          types.String `tfsdk:"name"`
	ConfigFile    types.String `tfsdk:"config_file"`
	CacheFile     types.String `tfsdk:"cache_file"`
	NoBrowserOpen types.Bool   `tfsdk:"no_browser_open"`
	ClientID      types.String `tfsdk:"client_id"`
}

type addressTemplate struct {
	Find    types.String `tfsdk:"find"`
	Replace types.String `tfsdk:"replace"`
}
type addressOptions struct {
	Insecure    types.Bool `tfsdk:"insecure"`
	NoTLSVerify types.Bool `tfsdk:"no_tls_verify"`
}
type config struct {
	AddressOptions      types.Map         `tfsdk:"address_options"`
	Token               types.String      `tfsdk:"token"`
	NoCredentials       types.Bool        `tfsdk:"no_credentials"`
	Resolvers           types.Map         `tfsdk:"resolvers"`
	ResolversEnv        types.String      `tfsdk:"resolvers_env"`
	Domain              types.String      `tfsdk:"domain"`
	DomainEnv           types.String      `tfsdk:"domain_env"`
	AddressTemplate     types.Object      `tfsdk:"address_template"`
	AddressTemplateEnv  types.String      `tfsdk:"address_template_env"`
	ServiceAccount      types.Object      `tfsdk:"service_account"`
	ModuleName          types.String      `tfsdk:"module_name"`
	VersionedEphemerals types.Dynamic     `tfsdk:"versioned_ephemeral_values"`
	ParentID            types.String      `tfsdk:"parent_id"`
	Profile             types.Object      `tfsdk:"profile"`
	Timeout             duration.Duration `tfsdk:"timeout"`
	AuthTimeout         duration.Duration `tfsdk:"auth_timeout"`
	PerRetryTimeout     duration.Duration `tfsdk:"per_retry_timeout"`
	Retries             types.Int64       `tfsdk:"retries"`
}

func (p *internalProvider) SDK() *gosdk.SDK {
	return p.sdk
}

func (p *internalProvider) Schema(
	_ context.Context,
	_ provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"address_options": schema.MapNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"insecure": schema.BoolAttribute{
							Optional:    true,
							Description: "use plain http connection",
						},
						"no_tls_verify": schema.BoolAttribute{
							Optional:    true,
							Description: "don't verify TLS certificates",
						},
					},
				},
				Description: "Set specific options for each address. Use " +
					"\"\\*\" to set them for all addresses at once. Empty " +
					"options will result in default TLS connection for this " +
					"address, thus overriding \"\\*\".",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Description: "authenticate using this IAM token",
			},
			"no_credentials": schema.BoolAttribute{
				Optional:    true,
				Description: "do not authenticate",
			},
			"resolvers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "resolver map of type [pattern|service_id]->" +
					"address",
				MarkdownDescription: "resolver map of type " +
					"`[pattern|service_id]->address`",
			},
			"resolvers_env": schema.StringAttribute{
				Optional: true,
				Description: "env variable name that holds resolver map (may" +
					" be set alongside resolvers)",
			},
			"domain": schema.StringAttribute{
				Optional:    true,
				Description: "custom domain name (overrides domain_env)",
			},
			"domain_env": schema.StringAttribute{
				Optional:    true,
				Description: "env variable name to obtain custom domain name",
			},
			"timeout": schema.StringAttribute{
				Optional:   true,
				CustomType: wellknown.WellKnownByName("google.protobuf.Duration").Type().(basetypes.StringTypable),
				Description: fmt.Sprintf(
					"timeout for each Nebius SDK request, default %s,"+
						" as a string: possibly signed sequence of decimal "+
						"numbers, each with optional fraction and a unit "+
						"suffix, such as `300ms`, `-1.5h` or `2h45m`. "+
						"Valid time units are `ns`, `us` (or `µs`), `ms`"+
						", `s`, `m`, `h`, `d`",
					gosdk.DefaultTimeout.String(),
				),
			},
			"auth_timeout": schema.StringAttribute{
				Optional:   true,
				CustomType: wellknown.WellKnownByName("google.protobuf.Duration").Type().(basetypes.StringTypable),
				Description: fmt.Sprintf(
					"timeout for each Nebius SDK request including "+
						"authentication, default %s, as a string: possibly signed sequence "+
						"of decimal numbers, each with optional fraction and a "+
						"unit suffix, such as `300ms`, `-1.5h` or `2h45m`. Valid "+
						"time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, "+
						"`h`, `d`",
					gosdk.DefaultAuthTimeout.String(),
				),
			},
			"per_retry_timeout": schema.StringAttribute{
				Optional:   true,
				CustomType: wellknown.WellKnownByName("google.protobuf.Duration").Type().(basetypes.StringTypable),
				Description: fmt.Sprintf(
					"timeout for each Nebius SDK request retry, default %s, as"+
						" a string: possibly signed sequence of decimal numbers,"+
						" each with optional fraction and a unit suffix, such as "+
						"`300ms`, `-1.5h` or `2h45m`. Valid time units are `ns`, "+
						"`us` (or `µs`), `ms`, `s`, `m`, `h`, `d`",
					gosdk.DefaultPerRetry.String(),
				),
			},
			"retries": schema.Int64Attribute{
				Optional: true,
				Description: fmt.Sprintf(
					"number of retries for each Nebius SDK request, default %d",
					gosdk.DefaultRetries,
				),
			},
			"address_template": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"find": schema.StringAttribute{
						Required: true,
					},
					"replace": schema.StringAttribute{
						Required: true,
					},
				},
				Optional: true,
				Description: "address template (overrides " +
					"address_template_env)",
			},
			"address_template_env": schema.StringAttribute{
				Optional: true,
				Description: "env variable name to obtain address template in" +
					" form of FIND=REPLACE",
				MarkdownDescription: "env variable name to obtain address " +
					"template in form of `FIND=REPLACE`",
			},
			"module_name": schema.StringAttribute{
				Optional: true,
				Description: "it is suggested to set this value to your " +
					"module name if the provider is initialized in one, does " +
					"not affect any behaviors",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-zA-Z0-9_]{0,16}$"),
						"must be a string of [a-zA-Z0-9_], not more than "+
							"16 characters",
					),
				},
			},
			"service_account": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"credentials_file": schema.StringAttribute{
						Optional: true,
						Description: "file path of the service account " +
							"credentials file (overrides credentials_file_env)" +
							"\nYou can set this file instead of other " +
							"parameters here, it will silently override them.",
					},
					"credentials_file_env": schema.StringAttribute{
						Optional: true,
						Description: "env var containing file path of the " +
							"service account credentials, same as " +
							"credentials_file",
					},
					"private_key": schema.StringAttribute{
						Optional: true,
						Description: "private key (overrides private_key_file" +
							" and private_key_file_env)",
					},
					"private_key_file": schema.StringAttribute{
						Optional: true,
						Description: "file path of the service account " +
							"private key (overrides private_key_file_env)",
					},
					"private_key_file_env": schema.StringAttribute{
						Optional: true,
						Description: "env var containing file path of the " +
							"service account private key",
					},
					"public_key_id": schema.StringAttribute{
						Optional: true,
						Description: "service account public key ID (" +
							"overrides public_key_id_env)",
					},
					"public_key_id_env": schema.StringAttribute{
						Optional: true,
						Description: "env var containing service account " +
							"public key ID",
					},
					"account_id": schema.StringAttribute{
						Optional: true,
						Description: "service account ID (overrides " +
							"account_id_env)",
					},
					"account_id_env": schema.StringAttribute{
						Optional:    true,
						Description: "env var containing service account ID",
					},
				},
				Description: "sets service account credentials (is overridden" +
					" by token)",
			},
			"parent_id": schema.StringAttribute{
				Optional: true,
				Description: "Parent ID if it is not read from the profile, " +
					"or if you want to overwrite it.",
			},
			"profile": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Optional: true,
						Description: "Profile name to use, uses the default " +
							"profile if not set.",
					},
					"config_file": schema.StringAttribute{
						Optional: true,
						Description: "File path to cache the token in, " +
							"defaults to `~/" + paths.DefaultConfigDir +
							"/config.yaml`",
					},
					"cache_file": schema.StringAttribute{
						Optional: true,
						Description: "File path to cache the token in, " +
							"defaults to `~/" + paths.DefaultConfigDir +
							"/credentials.yaml`",
					},
					"no_browser_open": schema.BoolAttribute{
						Optional: true,
						Description: "If set to true, the provider will not " +
							"open a browser window for federation " +
							"authentication, only showing the URL through the" +
							" logger.",
					},
					"client_id": schema.StringAttribute{
						Optional: true,
						Description: "Client ID for federation authentication" +
							", defaults to `" + defaultClientID + "`",
					},
				},
				Optional:            true,
				MarkdownDescription: "Reads profile from the CLI config.",
			},
		},
	}
	if os.Getenv(disableDynamicEnv) != "true" && p.WriteOnlyFieldsSupported() {
		resp.Schema.Attributes["versioned_ephemeral_values"] = schema.DynamicAttribute{
			Optional:  true,
			Sensitive: true,
			Description: "A map of all ephemeral values that will be " +
				"passed to nebius_hash for hashing",
		}
	}
}

func isKnown(val attr.Value) bool {
	return !val.IsNull() && !val.IsUnknown()
}

func parseAddressOptions(ctx context.Context, opts types.Map) (
	[]grpc.DialOption, diag.Diagnostics,
) {
	diags := diag.Diagnostics{}
	dialOpts := []grpc.DialOption{}
	wildCardCreds := credentials.NewTLS(nil)

	if isKnown(opts) {
		var parsed map[string]addressOptions
		diags.Append(opts.ElementsAs(ctx, &parsed, false)...)
		for addr, options := range parsed {
			creds := credentials.NewTLS(nil)
			if isKnown(options.Insecure) {
				creds = insecure.NewCredentials()
				if isKnown(options.NoTLSVerify) {
					diags.AddWarning("both insecure and no_tls_verify set",
						fmt.Sprintf(
							"both insecure and no_tls_verify set for %q, "+
								"insecure takes precedense", addr,
						),
					)
				}
			} else if isKnown(options.NoTLSVerify) {
				creds = credentials.NewTLS(&tls.Config{
					InsecureSkipVerify: true,
				})
			}
			if addr == "*" {
				wildCardCreds = creds
			} else {
				dialOpts = append(
					dialOpts,
					conn.WithAddressDialOptions(
						conn.Address(addr), grpc.WithTransportCredentials(
							creds,
						),
					),
				)
			}
		}
	}
	dialOpts = append(
		dialOpts,
		grpc.WithTransportCredentials(wildCardCreds),
	)
	return dialOpts, diags
}

func parseServiceAccount(ctx context.Context, saAttr types.Object) (
	auth.ServiceAccountReader, diag.Diagnostics,
) {
	var sa saConfig
	diags := diag.Diagnostics{}
	dg := saAttr.As(ctx, &sa, basetypes.ObjectAsOptions{})
	diags.Append(dg...)
	if !dg.HasError() {
		credentialsFileName := ""
		pkText := ""
		fileName := ""
		keyID := ""
		accountID := ""
		if isKnown(sa.CredentialsFile) {
			credentialsFileName = sa.CredentialsFile.ValueString()
		} else if isKnown(sa.CredentialsFileEnv) {
			credentialsFileName = os.Getenv(
				sa.CredentialsFileEnv.ValueString(),
			)
		}
		if credentialsFileName != "" {
			return auth.NewServiceAccountCredentialsFileParser(
				nil, credentialsFileName,
			), diags
		}

		if isKnown(sa.PrivateKey) {
			pkText = sa.PrivateKey.ValueString()
		} else if isKnown(sa.PrivateKeyFile) {
			fileName = sa.PrivateKeyFile.ValueString()
		} else if isKnown(sa.PrivateKeyFileEnv) {
			fileName = os.Getenv(sa.PrivateKeyFileEnv.ValueString())
		}
		if pkText == "" && fileName == "" {
			diags.AddError(
				"no service account private key provided",
				"Service account private key is required to set"+
					" up service account. Either provide "+
					"service_account.private_key, or provide "+
					"service_account.private_key_file, or "+
					"service_account.private_key_file_env and set that "+
					"env var accordingly",
			)
		}
		if isKnown(sa.PublicKeyID) {
			keyID = sa.PublicKeyID.ValueString()
		} else if isKnown(sa.PublicKeyIDEnv) {
			keyID = os.Getenv(sa.PublicKeyIDEnv.ValueString())
		}
		if keyID == "" {
			diags.AddError(
				"no service account public key ID provided",
				"Service account public key ID is required to set"+
					" up service account. Either provide "+
					"service_account.public_key_id or provide "+
					"service_account.public_key_id_env and set that "+
					"env var",
			)
		}
		if isKnown(sa.AccountID) {
			accountID = sa.AccountID.ValueString()
		} else if isKnown(sa.AccountIDEnv) {
			accountID = os.Getenv(sa.AccountIDEnv.ValueString())
		}
		if accountID == "" {
			diags.AddError(
				"no service account ID provided",
				"Service account ID is required to set"+
					" up service account. Either provide "+
					"service_account.account_id or provide "+
					"service_account.account_id_env and set that env var",
			)
		}
		if pkText != "" {
			return auth.NewPrivateKeyParser(
				[]byte(pkText),
				keyID,
				accountID,
			), diags
		}
		return auth.NewPrivateKeyFileParser(
			nil,
			fileName,
			keyID,
			accountID,
		), diags
	}
	return nil, diags
}

func (p *internalProvider) parseProfile(
	ctx context.Context,
	providerCfg config,
) (sdkconfig.ConfigInterface, diag.Diagnostics) {
	if !isKnown(providerCfg.Profile) {
		return nil, diag.Diagnostics{}
	}

	var profile profileConfig
	diags := diag.Diagnostics{}
	dg := providerCfg.Profile.As(ctx, &profile, basetypes.ObjectAsOptions{})
	diags.Append(dg...)
	if dg.HasError() {
		return nil, diags
	}

	opts := []sdkconfig.Option{
		reader.WithClientID(defaultClientID),
		reader.WithLogger(slog.New(&slogHandler{})), // debug logs before sdk logger is set
	}
	if isKnown(profile.Name) {
		opts = append(opts, reader.WithProfileName(profile.Name.ValueString()))
	}
	if isKnown(profile.ConfigFile) {
		opts = append(opts, reader.WithPath(profile.ConfigFile.ValueString()))
	}
	if isKnown(profile.CacheFile) {
		opts = append(opts, reader.WithCacheFileName(profile.CacheFile.ValueString()))
	}
	if isKnown(profile.NoBrowserOpen) {
		opts = append(opts, reader.WithNoBrowserOpen(profile.NoBrowserOpen.ValueBool()))
	}
	if isKnown(profile.ClientID) {
		opts = append(opts, reader.WithClientID(profile.ClientID.ValueString()))
	}

	cfg := reader.NewConfigReader(opts...)
	if err := cfg.Load(ctx); err != nil {
		diags.AddError(
			"failed to load config",
			fmt.Sprintf("failed to load config: %v", err),
		)
		return nil, diags
	}
	p.parentID = cfg.ParentID()
	return cfg, diags
}

func isInfoEnabled(level string) bool {
	switch level {
	case "TRACE":
		return true
	case "JSON": // same as TRACE, but in JSON format
		return true
	case "DEBUG":
		return true
	case "INFO":
		return true
	default:
		return false
	}
}

func (p *internalProvider) isInfoLogEnabled() bool {
	if isInfoEnabled(os.Getenv("TF_LOG")) {
		return true
	}
	if isInfoEnabled(os.Getenv("TF_LOG_PROVIDER")) {
		return true
	}
	nameUpper := strings.ToUpper(p.name())
	return isInfoEnabled(os.Getenv("TF_LOG_PROVIDER_" + nameUpper))
}

func (p *internalProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var data config
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	ver, err := version.BuildVersion()
	if err != nil {
		resp.Diagnostics.AddWarning("Failed to get provider build version",
			err.Error())
	}

	uaComments := []string{}
	if isKnown(data.ModuleName) {
		uaComments = append(uaComments, fmt.Sprintf("%q", data.ModuleName.ValueString()))
	}
	uaComments = append(uaComments, fmt.Sprintf("terraform/%s", req.TerraformVersion))

	userAgent := fmt.Sprintf(
		"terraform-provider-%s/%s (%s)",
		p.name(), ver, strings.Join(uaComments, "; "),
	)
	tflog.Info(ctx, "user-agent", map[string]any{
		"user_agent": userAgent,
	})
	retryOptions := []retry.CallOption{}
	if isKnown(data.Retries) {
		retryCount := data.Retries.ValueInt64()
		if retryCount <= 0 {
			resp.Diagnostics.AddError(
				"Invalid retry count",
				"Retry count must be a positive integer",
			)
		} else {
			retryOptions = append(retryOptions, retry.WithMax(uint(retryCount)))
		}
	}
	if isKnown(data.PerRetryTimeout) {
		perRetryTimeout, innerDiag := data.PerRetryTimeout.ValueDuration()
		if innerDiag.HasError() {
			resp.Diagnostics.Append(innerDiag...)
		} else {
			retryOptions = append(retryOptions, retry.WithPerRetryTimeout(perRetryTimeout.AsDuration()))
		}
	}
	dialOptions := []grpc.DialOption{}
	options := []gosdk.Option{
		gosdk.WithRetryOptions(retryOptions...),
	}
	resolvers := []conn.Resolver{}
	options = append(options, gosdk.WithUserAgentPrefix(userAgent))

	if isKnown(data.Timeout) {
		timeout, innerDiag := data.Timeout.ValueDuration()
		if innerDiag.HasError() {
			resp.Diagnostics.Append(innerDiag...)
		} else {
			options = append(options, gosdk.WithTimeout(timeout.AsDuration()))
		}
	}
	if isKnown(data.AuthTimeout) {
		authTimeout, innerDiag := data.AuthTimeout.ValueDuration()
		if innerDiag.HasError() {
			resp.Diagnostics.Append(innerDiag...)
		} else {
			options = append(options, gosdk.WithAuthTimeout(authTimeout.AsDuration()))
		}
	}

	cfgReader, innerDiag := p.parseProfile(
		ctx,
		data,
	)
	resp.Diagnostics.Append(innerDiag...)
	if innerDiag.HasError() {
		return
	}

	var tokener auth.BearerTokener
	if isKnown(data.NoCredentials) && data.NoCredentials.ValueBool() {
		options = append(options, gosdk.WithCredentials(
			gosdk.NoCredentials(),
		))
		tflog.Debug(ctx, "Using provider without credentials")
	} else if isKnown(data.Token) {
		tokener = auth.StaticBearerToken(data.Token.ValueString())
		tflog.Debug(ctx, "Using provider with static token")
	} else if isKnown(data.ServiceAccount) {
		tflog.Debug(ctx, "Using provider with Service Account")
		opt, innerDiag := parseServiceAccount(ctx, data.ServiceAccount)
		resp.Diagnostics.Append(innerDiag...)
		tokener = auth.NewExchangeableBearerTokenerWithDeferredClient(
			auth.NewServiceAccountExchangeTokenRequester(
				opt,
			),
			func() (iampb.TokenExchangeServiceClient, error) {
				if p.sdk == nil {
					return nil, errors.New("SDK is not initialized")
				}
				return iam.NewTokenExchangeService(p.sdk), nil
			},
		)
	} else {
		if token := os.Getenv(constants.TokenEnv); token != "" && cfgReader == nil {
			tokener = auth.StaticBearerToken(token)
			tflog.Debug(ctx, "Using provider with token")
		}
		if cfgReader != nil {
			if cfgReader.GetAuthType() == sdkconfig.AuthTypeFederation &&
				!p.isInfoLogEnabled() {
				nameUpper := strings.ToUpper(p.name())
				resp.Diagnostics.AddWarning(
					"using federation authentication",
					"Using federation authentication from profile "+
						cfgReader.CurrentProfileName()+", "+
						"the provider may open a browser window "+
						"to authenticate or require navigating to a "+
						"particular URL. You have to enable logs to see it, at"+
						" least at the INFO level, by setting the environment"+
						" variable TF_LOG, TF_LOG_PROVIDER or TF_LOG_PROVIDER_"+
						nameUpper+" to INFO, or use a service account "+
						"with the terraform provider instead.",
				)
			}
		}
		if os.Getenv(constants.TokenEnv) == "" && !isKnown(data.Profile) {
			tflog.Warn(
				ctx,
				"No credentials provided, using provider without "+
					"authorization. Did you forget to set "+constants.TokenEnv+
					" env variable?",
			)
			options = append(options, gosdk.WithCredentials(
				gosdk.NoCredentials(),
			))
		}
	}
	if cfgReader != nil {
		options = append(options,
			gosdk.WithConfigReader(cfgReader),
		)
	}

	if tokener != nil {

		options = append(options, gosdk.WithCredentials(
			gosdk.CustomTokener(tokener),
		))
	}

	if isKnown(data.ParentID) { // must be after parseProfile
		p.parentID = data.ParentID.ValueString()
	}
	if isKnown(data.Resolvers) {
		for k, v := range data.Resolvers.Elements() {
			if isKnown(v) {
				addr, _ := v.(types.String)
				resolvers = append(
					resolvers,
					conn.NewBasicResolver(k, addr.ValueString()),
				)
			}
		}
	}
	overrideDialOpts := []grpc.DialOption{}
	if isKnown(data.ResolversEnv) {
		resolverFromEnv, dialOptionsFromEnv, err := conn.ParseResolverAndDialOptions(
			os.Getenv(data.ResolversEnv.ValueString()),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"failed to parse resolver from env",
				fmt.Sprintf("failed to parse resolver from env: %s", err),
			)
		} else {
			resolvers = append(resolvers, resolverFromEnv)
			overrideDialOpts = append(overrideDialOpts, dialOptionsFromEnv...)
		}
	}

	dialOptions = append(dialOptions, overrideDialOpts...)

	if len(resolvers) > 0 {
		options = append(options, gosdk.WithResolvers(resolvers...))
	}

	if isKnown(data.Domain) {
		options = append(options, gosdk.WithDomain(data.Domain.ValueString()))
		tflog.Debug(ctx, "Selected provider domain", map[string]any{
			"domain": data.Domain.ValueString(),
		})
		if isKnown(data.DomainEnv) {
			resp.Diagnostics.AddWarning(
				"both domain and domain_env set",
				"both domain and domain_env settings set in config, skipping "+
					"domain_env",
			)
		}
	} else if isKnown(data.DomainEnv) {
		options = append(options, gosdk.WithDomain(os.Getenv(
			data.DomainEnv.ValueString(),
		)))
		tflog.Debug(ctx, "Selected provider domain", map[string]any{
			"domain": os.Getenv(
				data.DomainEnv.ValueString(),
			),
		})
	} else if cfgReader != nil && cfgReader.Endpoint() != "" {
		tflog.Debug(ctx, "Selected provider domain from config",
			map[string]any{
				"domain":  cfgReader.Endpoint(),
				"profile": cfgReader.CurrentProfileName(),
			})
	} else {
		options = append(options, gosdk.WithDomain(constants.Domain))
		tflog.Debug(ctx, "Selected provider domain", map[string]any{
			"domain": constants.Domain,
		})
	}

	if isKnown(data.AddressTemplate) {
		var tpl addressTemplate
		dg := data.AddressTemplate.As(ctx, &tpl, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(dg...)
		if !dg.HasError() {
			if isKnown(tpl.Find) && isKnown(tpl.Replace) {
				options = append(options, gosdk.WithAddressTemplate(
					tpl.Find.ValueString(),
					tpl.Replace.ValueString(),
				))
				if isKnown(data.AddressTemplateEnv) {
					resp.Diagnostics.AddWarning(
						"both address_template and address_template_env set",
						"both address_template and address_template_env "+
							"settings set in config, skipping "+
							"address_template_env",
					)
				}
			}
		}
	} else if isKnown(data.AddressTemplateEnv) {
		tpl := os.Getenv(data.AddressTemplateEnv.ValueString())
		if tpl != "" {
			parts := strings.SplitN(tpl, "=", 1)
			if len(parts) < 2 {
				resp.Diagnostics.AddError(
					"\"=\" not found inside env variable",
					fmt.Sprintf("\"=\" not found inside env variable: %q", tpl),
				)
			} else {
				options = append(options, gosdk.WithAddressTemplate(
					parts[0], parts[1],
				))
			}
		}
	}

	// double isKnown — in the future if block removed
	if isKnown(data.AddressOptions) {
		addressDialOptions, innerDiag := parseAddressOptions(
			ctx, data.AddressOptions,
		)
		resp.Diagnostics.Append(innerDiag...)
		dialOptions = append(dialOptions, addressDialOptions...)
	} else { // TODO: remove deprecated options in the future
		withTLS := grpc.WithTransportCredentials(credentials.NewTLS(nil))
		dialOptions = append(dialOptions, withTLS)
	}

	if len(dialOptions) > 0 {
		options = append(options, gosdk.WithDialOptions(dialOptions...))
	}

	options = append(options, gosdk.WithLoggerHandler(&slogHandler{}))

	// terraform must set everything explicitly
	// use `nebius_parent_id` data source to obtain the default parent ID
	options = append(options, gosdk.WithoutParentID())

	sdk, err := gosdk.New(ctx, options...)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to make gosdk",
			fmt.Sprintf("failed to make gosdk: %s", err),
		)
	} else {
		p.sdk = sdk
	}

	if isKnown(data.VersionedEphemerals) {
		val, _, diags := ctypes.UnwrapDynamic(ctx, data.VersionedEphemerals)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		tfVal, err := data.VersionedEphemerals.ToTerraformValue(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"failed to convert versioned_ephemeral_values to terraform value",
				fmt.Sprintf("failed to convert versioned_ephemeral_values to terraform value: %s", err),
			)
		}
		if !tfVal.IsFullyKnown() {
			resp.Diagnostics.AddError(
				"versioned_ephemeral_values is not fully known",
				"versioned_ephemeral_values must be fully known",
			)
		} else {
			ephMap, ok := val.(types.Object)
			if !ok {
				resp.Diagnostics.AddError(
					"versioned_ephemeral_values is not an object",
					fmt.Sprintf(
						"versioned_ephemeral_values is %s, object (map of anything) required",
						val.Type(ctx),
					),
				)
			}
			p.versionedEphemerals = ephMap.Attributes()
		}
	}
}

func (p *internalProvider) WriteOnlyFieldsSupported() bool {
	return os.Getenv(disableWriteOnlyEnv) != "true"
}

func (p *internalProvider) name() string {
	return Name
}

func (p *internalProvider) Metadata(
	_ context.Context,
	_ provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = p.name()
}

func (p *internalProvider) DataSources(
	_ context.Context,
) []func() datasource.DataSource {
	ret := []func() datasource.DataSource{
		func() datasource.DataSource {
			return api.NewParentID(p.parentID)
		},
	}
	for _, f := range nebius.DatasourceFactories {
		ret = append(ret, func() datasource.DataSource {
			return f(p)
		})
	}
	return ret
}

func (p *internalProvider) Resources(
	_ context.Context,
) []func() resource.Resource {
	ret := []func() resource.Resource{}

	if os.Getenv(disableDynamicEnv) != "true" &&
		p.WriteOnlyFieldsSupported() {
		ret = append(ret, func() resource.Resource {
			return api.NewHashResource(p.versionedEphemerals)
		})
	}
	for _, f := range nebius.ResourceFactories {
		ret = append(ret, func() resource.Resource {
			return f(p)
		})
	}
	return ret
}

func (p *internalProvider) EphemeralResources(
	_ context.Context,
) []func() ephemeral.EphemeralResource {
	if !p.WriteOnlyFieldsSupported() {
		return nil // no ephemeral resources if write-only is disabled
	}
	ret := []func() ephemeral.EphemeralResource{
		token.NewIAMTokenFactory(p),
	}
	for _, f := range nebius.EphemeralFactories {
		ret = append(ret, func() ephemeral.EphemeralResource {
			return f(p)
		})
	}
	return ret
}
