package main

import (
    "context"
    "net/http"
    "os"
    "time"

    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-framework/path"
    // Add structured logging
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the provider satisfies the expected interfaces. The provider
// must implement the provider.Provider interface.  The framework
// enforces these interfaces at compile time.
var _ provider.Provider = &tenablevmProvider{}

// tenablevmProvider models the Terraform provider implementation.  It
// holds the version string which is set when building the plugin.
// Providers may maintain internal state across requests, but this
// implementation does not currently need it.
type tenablevmProvider struct {
    version string
}

// NewProvider returns a new instance of the Tenable VM provider with
// the supplied version.  This function is referenced by the main
// package to create the provider server.  When publishing the
// provider, the version should be replaced by the build tooling.
func NewProvider(version string) provider.Provider {
    return &tenablevmProvider{
        version: version,
    }
}

// Metadata returns the provider type name and version.  The type name
// becomes the namespace for resources and data sources (e.g.
// tenablevm_user).  The version is surfaced in provider logs and
// diagnostics.  See the framework documentation for more details
//【718857133965766†L690-L731】.
func (p *tenablevmProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "tenablevm"
    resp.Version = p.version
}

// tenableProviderModel maps provider configuration schema data into a
// Go struct.  The `tfsdk` struct tags correspond to the schema
// attribute names.  All fields are defined as types.String to take
// advantage of the framework's null/unknown semantics.
type tenableProviderModel struct {
    AccessKey types.String `tfsdk:"access_key"`
    SecretKey types.String `tfsdk:"secret_key"`
}

// Schema defines the provider-level configuration schema. The provider
// accepts optional access_key and secret_key attributes (falling back to
// environment variables). Sensitive fields are marked accordingly so
// they are redacted from logs and state. Defaults are handled in
// Configure.
func (p *tenablevmProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "access_key": schema.StringAttribute{
                Optional:    true,
                Sensitive:   false,
                Description: "Tenable Vulnerability Management API access key. Can also be provided via the TENABLE_ACCESS_KEY environment variable.",
            },
            "secret_key": schema.StringAttribute{
                Optional:    true,
                Sensitive:   true,
                Description: "Tenable Vulnerability Management API secret key. Can also be provided via the TENABLE_SECRET_KEY environment variable.",
            },
        },
        Description: "The Tenable VM provider configures access to the Tenable Vulnerability Management API.",
    }
}

// Configure prepares a Tenable VM API client for data sources and
// resources.  It reads the provider configuration, applies
// environment variable fallbacks, validates required fields, and
// instantiates the client.  On error, diagnostics are appended to
// resp.Diagnostics.  On success, the client is stored in
// resp.ResourceData and resp.DataSourceData for use by resources and
// data sources【718857133965766†L747-L872】.
func (p *tenablevmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    // Retrieve provider data from configuration into a model struct
    var config tenableProviderModel
    diags := req.Config.Get(ctx, &config)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Check for unknown values and raise attribute errors
    if config.AccessKey.IsUnknown() {
        resp.Diagnostics.AddAttributeError(
            path.Root("access_key"),
            "Unknown Tenable API Access Key",
            "The provider cannot create the Tenable API client because there is an unknown value for the access_key. Either set the value directly in the configuration, or use the TENABLE_ACCESS_KEY environment variable.",
        )
    }
    if config.SecretKey.IsUnknown() {
        resp.Diagnostics.AddAttributeError(
            path.Root("secret_key"),
            "Unknown Tenable API Secret Key",
            "The provider cannot create the Tenable API client because there is an unknown value for the secret_key. Either set the value directly in the configuration, or use the TENABLE_SECRET_KEY environment variable.",
        )
    }
    if resp.Diagnostics.HasError() {
        return
    }

    // Default values to environment variables, override with config if provided
    accessKey := os.Getenv("TENABLE_ACCESS_KEY")
    secretKey := os.Getenv("TENABLE_SECRET_KEY")

    if !config.AccessKey.IsNull() {
        accessKey = config.AccessKey.ValueString()
    }
    if !config.SecretKey.IsNull() {
        secretKey = config.SecretKey.ValueString()
    }

    // Validate required credentials
    if accessKey == "" {
        resp.Diagnostics.AddAttributeError(
            path.Root("access_key"),
            "Missing Tenable API access key",
            "An access_key must be provided either in the configuration or via the TENABLE_ACCESS_KEY environment variable.",
        )
    }
    if secretKey == "" {
        resp.Diagnostics.AddAttributeError(
            path.Root("secret_key"),
            "Missing Tenable API secret key",
            "A secret_key must be provided either in the configuration or via the TENABLE_SECRET_KEY environment variable.",
        )
    }
    if resp.Diagnostics.HasError() {
        return
    }

    // Structured logging: set log fields for credentials (mask secret key).
    // Use tflog.SetField to store context-specific fields which will be included in
    // subsequent log messages. Mask sensitive information using MaskFieldValuesWithFieldKeys.
    ctx = tflog.SetField(ctx, "tenable_access_key", accessKey)
    ctx = tflog.SetField(ctx, "tenable_secret_key", secretKey)
    ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "tenable_secret_key")

    // Log a debug message before constructing the API client【301259032402045†L324-L365】.
    tflog.Debug(ctx, "Creating Tenable VM client")

    // Construct the HTTP client with a reasonable timeout
    httpClient := &http.Client{Timeout: 60 * time.Second}
    apiClient := &Client{
        AccessKey: accessKey,
        SecretKey: secretKey,
        Http:      httpClient,
    }

    // Tenable does not provide a lightweight endpoint to validate
    // credentials without side effects.  As such, we assume the
    // credentials are valid and defer any errors to resource CRUD
    // operations.  Diagnostics generated during those operations will
    // surface to the practitioner.

    // Make the Tenable client available to resources and data sources
    resp.ResourceData = apiClient
    resp.DataSourceData = apiClient

    // Log an info message indicating successful configuration【301259032402045†L324-L365】.
    tflog.Info(ctx, "Configured Tenable VM client", map[string]any{"success": true})
}

// Resources defines the resources implemented in this provider.  The
// returned slice contains factory functions which instantiate new
// resource types on demand.  In this provider we expose a single
// resource for managing Tenable VM users.
func (p *tenablevmProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewUserResource,
    }
}

// DataSources defines the data sources implemented in this provider.  The
// Tenable VM provider currently does not implement any data sources.
func (p *tenablevmProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        NewUserDataSource,
        NewRoleDataSource,
        NewGroupDataSource,
    }
}