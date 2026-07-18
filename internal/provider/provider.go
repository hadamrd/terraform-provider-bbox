// Package provider implements the Terraform provider for the Bouygues Bbox
// router admin API, backed by github.com/hadamrd/bbox-cli/pkg/client.
package provider

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/hadamrd/bbox-cli/pkg/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SharedClient is the typed handle Configure hands to resources / data sources.
type SharedClient struct {
	Client *client.Client
}

// bboxProvider is the framework provider implementation.
type bboxProvider struct {
	version string
}

// New returns a provider constructor for providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &bboxProvider{version: version}
	}
}

// providerModel maps the user-facing schema into Go.
type providerModel struct {
	SessionFile  types.String `tfsdk:"session_file"`
	PasswordFile types.String `tfsdk:"password_file"`
	BaseURL      types.String `tfsdk:"base_url"`
	Retries      types.Int64  `tfsdk:"retries"`
	Timeout      types.String `tfsdk:"timeout"`
}

func (p *bboxProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bbox"
	resp.Version = p.version
}

func (p *bboxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a Bouygues Bbox router via its reversed admin API.",
		Attributes: map[string]schema.Attribute{
			"session_file": schema.StringAttribute{
				Optional:    true,
				Description: "Path to cached session cookies. Defaults to ~/.bbox-session.json. Env: BBOX_SESSION_FILE.",
			},
			"password_file": schema.StringAttribute{
				Optional:    true,
				Description: "Path to the admin password file. Defaults to ~/.bbox-password. Env: BBOX_PASSWORD_FILE.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Router base URL. Default https://mabbox.bytel.fr. Env: BBOX_BASE_URL.",
			},
			"retries": schema.Int64Attribute{
				Optional:    true,
				Description: "Retry count for transient network errors. Default 2. Env: BBOX_RETRIES.",
			},
			"timeout": schema.StringAttribute{
				Optional:    true,
				Description: "HTTP timeout as a Go duration string (e.g. \"15s\"). Env: BBOX_TIMEOUT.",
			},
		},
	}
}

func (p *bboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionFile := pickString(cfg.SessionFile, "BBOX_SESSION_FILE", client.SessionFile())
	passwordFile := pickString(cfg.PasswordFile, "BBOX_PASSWORD_FILE", client.PasswordFileDefault())
	baseURL := pickString(cfg.BaseURL, "BBOX_BASE_URL", client.BaseURL)
	retries := pickInt(cfg.Retries, "BBOX_RETRIES", 2)
	timeout := pickDuration(cfg.Timeout, "BBOX_TIMEOUT", 15*time.Second)

	// bbox-cli stores BaseURL / SessionFile as package globals; honour the overrides
	// by writing them back before constructing the client. Session lookups performed
	// during EnsureAuth will observe the values set here.
	client.BaseURL = baseURL
	// Note: SessionFile() returns a computed default; there is no setter, so a
	// non-default session_file is only meaningful once bbox-cli exposes one. For now
	// we surface the config value via env so downstream tooling can read it.
	_ = os.Setenv("BBOX_SESSION_FILE", sessionFile)

	c := client.New(false, retries, timeout)

	// Password getter: BBOX_PASSWORD env wins, then the file. Nothing is prompted —
	// Terraform runs are non-interactive.
	pwGetter := func() (string, error) {
		if v := os.Getenv("BBOX_PASSWORD"); v != "" {
			return v, nil
		}
		b, err := os.ReadFile(passwordFile)
		if err != nil {
			return "", err
		}
		return trimTrailingNewline(string(b)), nil
	}
	c.WithPasswordGetter(pwGetter)

	if err := c.EnsureAuth(pwGetter); err != nil {
		resp.Diagnostics.AddError(
			"Bbox authentication failed",
			"Could not authenticate to the Bbox router: "+err.Error(),
		)
		return
	}

	shared := &SharedClient{Client: c}
	resp.ResourceData = shared
	resp.DataSourceData = shared
}

func (p *bboxProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *bboxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// pickString applies env-then-default fallback for a string attribute.
func pickString(attr types.String, env, def string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

func pickInt(attr types.Int64, env string, def int) int {
	if !attr.IsNull() && !attr.IsUnknown() {
		return int(attr.ValueInt64())
	}
	if v := os.Getenv(env); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func pickDuration(attr types.String, env string, def time.Duration) time.Duration {
	raw := ""
	if !attr.IsNull() && !attr.IsUnknown() {
		raw = attr.ValueString()
	}
	if raw == "" {
		raw = os.Getenv(env)
	}
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}

func trimTrailingNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
