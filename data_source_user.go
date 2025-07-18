package main

import (
    "context"
    "strconv"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
    // Structured logging for data source
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

// userDataSource implements a data source for retrieving information about
// an existing Tenable VM user.  The data source accepts either the
// numeric user ID or the username as input and returns the user's
// attributes.  This allows configuration to reference users that are
// managed outside of Terraform or created by other processes.
//
// Example usage in a Terraform configuration:
//
// ```hcl
// data "tenablevm_user" "example" {
//   username = "alice@example.com"
// }
//
// output "tenable_user_id" {
//   value = data.tenablevm_user.example.id
// }
// ```
//
// Either `id` or `username` must be specified.  If both are
// provided, `id` takes precedence.  If neither is provided, the
// data source will return an error.
type userDataSource struct {
    client *Client
}

// userDataSourceModel maps the data source schema into a Go struct.
// Attributes that are not provided in the configuration are ignored
// on input.  All attributes are computed on output.
type userDataSourceModel struct {
    ID          types.String `tfsdk:"id"`
    Username    types.String `tfsdk:"username"`
    Name        types.String `tfsdk:"name"`
    Email       types.String `tfsdk:"email"`
    Permissions types.Int64  `tfsdk:"permissions"`
    Enabled     types.Bool   `tfsdk:"enabled"`
}

// NewUserDataSource returns a new data source instance.  The provider
// calls this function when registering data sources.
func NewUserDataSource() datasource.DataSource {
    return &userDataSource{}
}

// Metadata sets the type name for the data source.  The type name is
// appended to the provider type name, producing `tenablevm_user`.
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the attributes accepted and returned by the data
// source.  Input attributes `id` and `username` are optional and
// computed so that the value not provided by the configuration will
// be filled in based on the lookup result.  The other attributes are
// computed and describe the resolved user.
func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Numeric identifier of the user.",
                MarkdownDescription: "Numeric identifier of the user.",
            },
            "username": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Username of the Tenable VM user.",
                MarkdownDescription: "Username of the Tenable VM user.",
            },
            "name": schema.StringAttribute{
                Computed:    true,
                Description: "Human‑readable name of the user.",
                MarkdownDescription: "Human‑readable name of the user.",
            },
            "email": schema.StringAttribute{
                Computed:    true,
                Description: "Email address of the user.",
                MarkdownDescription: "Email address of the user.",
            },
            "permissions": schema.Int64Attribute{
                Computed:    true,
                Description: "Permissions integer for the user. See Tenable's role documentation for valid values.",
                MarkdownDescription: "Permissions integer for the user. See Tenable's role documentation for valid values.",
            },
            "enabled": schema.BoolAttribute{
                Computed:    true,
                Description: "Whether the user account is enabled.",
                MarkdownDescription: "Whether the user account is enabled.",
            },
        },
        Description:         "Retrieves information about a Tenable VM user by ID or username.",
        MarkdownDescription: "Retrieves information about a Tenable VM user by ID or username.",
    }
}

// Configure stores the provider's API client on the data source.  The
// framework ensures this is called before Read.  If no provider data
// is supplied (e.g. during unit tests), the data source remains
// unconfigured.
func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Provider Data Type",
            "The provider data supplied to the tenablevm_user data source is not a *Client. This is a bug in the provider implementation.",
        )
        return
    }
    d.client = c
}

// Read performs the lookup operation.  It determines the search key
// based on the configuration, calls the appropriate client method,
// and populates the state with the resolved user attributes.  Errors
// encountered during the lookup are appended to the diagnostics.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // Log debug message at beginning of read
    tflog.Debug(ctx, "Reading Tenable VM user data source")

    var config userDataSourceModel
    // Read the configuration into the model.  Unknown values are ignored.
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...) 
    if resp.Diagnostics.HasError() {
        return
    }
    // Determine which key to use for lookup.  id has precedence.
    var user *User
    if !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != "" {
        idStr := config.ID.ValueString()
        id, err := strconv.Atoi(idStr)
        if err != nil {
            resp.Diagnostics.AddAttributeError(
                path.Root("id"),
                "Invalid ID",
                "The id attribute must be a numeric string.",
            )
            return
        }
        u, err := d.client.GetUser(id)
        if err != nil {
            resp.Diagnostics.AddError(
                "Error retrieving Tenable VM user",
                err.Error(),
            )
            return
        }
        user = u
    } else if !config.Username.IsNull() && !config.Username.IsUnknown() && config.Username.ValueString() != "" {
        username := config.Username.ValueString()
        users, err := d.client.ListUsers()
        if err != nil {
            resp.Diagnostics.AddError(
                "Error listing Tenable VM users",
                err.Error(),
            )
            return
        }
        for _, u := range users {
            if u.Username == username {
                user = u
                break
            }
        }
        if user == nil {
            resp.Diagnostics.AddError(
                "User Not Found",
                "No Tenable VM user was found with username " + username + ".",
            )
            return
        }
    } else {
        resp.Diagnostics.AddError(
            "Missing Search Parameter",
            "Either the id or username attribute must be set to look up a Tenable VM user.",
        )
        return
    }
    // Populate state from the resolved user
    var state userDataSourceModel
    state.ID = types.StringValue(strconv.Itoa(user.ID))
    state.Username = types.StringValue(user.Username)
    if user.Name != "" {
        state.Name = types.StringValue(user.Name)
    } else {
        state.Name = types.StringNull()
    }
    if user.Email != "" {
        state.Email = types.StringValue(user.Email)
    } else {
        state.Email = types.StringNull()
    }
    state.Permissions = types.Int64Value(int64(user.Permissions))
    state.Enabled = types.BoolValue(user.Enabled)
    // Write computed state
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) 
    // Log info message with found user
    tflog.Info(ctx, "Read Tenable VM user data source", map[string]any{
        "user_id":  state.ID.ValueString(),
        "username": state.Username.ValueString(),
    })
}