package main

import (
    "context"
    "strconv"
    "strings"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/path"
    "github.com/hashicorp/terraform-plugin-framework/types"
    // Structured logging
    "github.com/hashicorp/terraform-plugin-log/tflog"
)

// roleDataSource implements a data source that retrieves a single Tenable VM
// role by ID or name.  Roles define sets of privileges that can be
// assigned to users or groups.  The underlying API does not provide
// a direct endpoint to retrieve a specific role by ID, so this data
// source calls ListRoles and filters the results.  Either `id` or
// `name` must be specified; if both are provided, `id` takes
// precedence.
type roleDataSource struct {
    client *Client
}

// roleDataSourceModel defines the state structure for the role data
// source.  All attributes are computed.  The id and name attributes
// are also optional inputs for filtering.
type roleDataSourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    UUID        types.String `tfsdk:"uuid"`
    Description types.String `tfsdk:"description"`
}

// NewRoleDataSource returns a new role data source.  The provider
// calls this function when registering data sources.
func NewRoleDataSource() datasource.DataSource {
    return &roleDataSource{}
}

// Metadata sets the data source type name.  The resulting type name
// will be `tenablevm_role`.
func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the input and output attributes for the role data
// source.  The id and name attributes are optional filters used to
// select a single role.  The uuid and description are computed.
func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Numeric identifier of the role. If set, this value is used to locate the role.",
                MarkdownDescription: "Numeric identifier of the role. If set, this value is used to locate the role.",
            },
            "name": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Name of the role. Used to locate the role when id is not provided.",
                MarkdownDescription: "Name of the role. Used to locate the role when id is not provided.",
            },
            "uuid": schema.StringAttribute{
                Computed:    true,
                Description: "UUID of the role.",
                MarkdownDescription: "UUID of the role.",
            },
            "description": schema.StringAttribute{
                Computed:    true,
                Description: "Description of the role.",
                MarkdownDescription: "Description of the role.",
            },
        },
        Description:         "Retrieves a Tenable VM role by ID or name.",
        MarkdownDescription: "Retrieves a Tenable VM role by ID or name.",
    }
}

// Configure stores the API client on the data source.
func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Provider Data Type",
            "The provider data supplied to the tenablevm_role data source is not a *Client. This is a bug in the provider implementation.",
        )
        return
    }
    d.client = c
}

// Read executes the lookup for a role by ID or name.  It calls
// ListRoles and filters the results.  If a matching role is found,
// the data source state is populated with the role's attributes.
func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // If the client is nil, do nothing.
    if d.client == nil {
        return
    }
    // Log debug message
    tflog.Debug(ctx, "Reading Tenable VM role data source")
    // Retrieve configuration into model
    var config roleDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...) // ignore unknown values
    if resp.Diagnostics.HasError() {
        return
    }
    // Determine search criteria: id takes precedence over name
    var role *Role
    if !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != "" {
        // parse ID string to int
        idStr := config.ID.ValueString()
        id, err := strconv.Atoi(idStr)
        if err != nil {
            resp.Diagnostics.AddAttributeError(
                path.Root("id"),
                "Invalid Role ID",
                "The id attribute must be a numeric string.",
            )
            return
        }
        // call ListRoles and find by ID
        roles, err := d.client.ListRoles()
        if err != nil {
            resp.Diagnostics.AddError(
                "Error listing Tenable VM roles",
                err.Error(),
            )
            return
        }
        for _, r := range roles {
            if r.ID == id {
                role = r
                break
            }
        }
        if role == nil {
            resp.Diagnostics.AddError(
                "Role Not Found",
                "No Tenable VM role was found with id " + idStr + ".",
            )
            return
        }
    } else if !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != "" {
        name := config.Name.ValueString()
        roles, err := d.client.ListRoles()
        if err != nil {
            resp.Diagnostics.AddError(
                "Error listing Tenable VM roles",
                err.Error(),
            )
            return
        }
        for _, r := range roles {
            if strings.EqualFold(r.Name, name) {
                role = r
                break
            }
        }
        if role == nil {
            resp.Diagnostics.AddError(
                "Role Not Found",
                "No Tenable VM role was found with name " + name + ".",
            )
            return
        }
    } else {
        resp.Diagnostics.AddError(
            "Missing Search Parameter",
            "Either the id or name attribute must be set to look up a Tenable VM role.",
        )
        return
    }
    // Build state from found role
    var state roleDataSourceModel
    state.ID = types.StringValue(strconv.Itoa(role.ID))
    state.Name = types.StringValue(role.Name)
    state.UUID = types.StringValue(role.UUID)
    if role.Description != "" {
        state.Description = types.StringValue(role.Description)
    } else {
        state.Description = types.StringNull()
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) 
    // Log info message with found role
    tflog.Info(ctx, "Read Tenable VM role data source", map[string]any{
        "role_id": state.ID.ValueString(),
        "name":    state.Name.ValueString(),
    })
}
