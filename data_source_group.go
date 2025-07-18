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

// groupDataSource implements a data source that retrieves a single Tenable VM
// group by ID or name.  Groups are used to manage collections of users
// and access.  The API does not provide a direct get-by-ID endpoint,
// so this data source calls ListGroups and filters the results.  Either
// `id` or `name` must be specified; if both are provided, `id` takes
// precedence.
type groupDataSource struct {
    client *Client
}

// groupDataSourceModel defines the state structure for the group data
// source.  All attributes are computed.  The id and name attributes
// are also optional inputs for filtering.
type groupDataSourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    UUID        types.String `tfsdk:"uuid"`
    Description types.String `tfsdk:"description"`
}

// NewGroupDataSource returns a new group data source.
func NewGroupDataSource() datasource.DataSource {
    return &groupDataSource{}
}

// Metadata sets the data source type name to `tenablevm_group`.
func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the input and output attributes for the group data
// source.  The id and name attributes are optional filters used to
// select a single group.
func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Numeric identifier of the group. If set, this value is used to locate the group.",
                MarkdownDescription: "Numeric identifier of the group. If set, this value is used to locate the group.",
            },
            "name": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Name of the group. Used to locate the group when id is not provided.",
                MarkdownDescription: "Name of the group. Used to locate the group when id is not provided.",
            },
            "uuid": schema.StringAttribute{
                Computed:    true,
                Description: "UUID of the group.",
                MarkdownDescription: "UUID of the group.",
            },
            "description": schema.StringAttribute{
                Computed:    true,
                Description: "Description of the group.",
                MarkdownDescription: "Description of the group.",
            },
        },
        Description:         "Retrieves a Tenable VM group by ID or name.",
        MarkdownDescription: "Retrieves a Tenable VM group by ID or name.",
    }
}

// Configure stores the API client on the data source.
func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Provider Data Type",
            "The provider data supplied to the tenablevm_group data source is not a *Client. This is a bug in the provider implementation.",
        )
        return
    }
    d.client = c
}

// Read executes the lookup for a group by ID or name.  It calls
// ListGroups and filters the results.  If a matching group is
// found, the data source state is populated with the group's
// attributes.
func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    if d.client == nil {
        return
    }
    // Log debug
    tflog.Debug(ctx, "Reading Tenable VM group data source")
    var config groupDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...) // ignore unknown values
    if resp.Diagnostics.HasError() {
        return
    }
    var group *Group
    if !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != "" {
        idStr := config.ID.ValueString()
        id, err := strconv.Atoi(idStr)
        if err != nil {
            resp.Diagnostics.AddAttributeError(
                path.Root("id"),
                "Invalid Group ID",
                "The id attribute must be a numeric string.",
            )
            return
        }
        groups, err := d.client.ListGroups()
        if err != nil {
            resp.Diagnostics.AddError(
                "Error listing Tenable VM groups",
                err.Error(),
            )
            return
        }
        for _, g := range groups {
            if g.ID == id {
                group = g
                break
            }
        }
        if group == nil {
            resp.Diagnostics.AddError(
                "Group Not Found",
                "No Tenable VM group was found with id " + idStr + ".",
            )
            return
        }
    } else if !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != "" {
        name := config.Name.ValueString()
        groups, err := d.client.ListGroups()
        if err != nil {
            resp.Diagnostics.AddError(
                "Error listing Tenable VM groups",
                err.Error(),
            )
            return
        }
        for _, g := range groups {
            if strings.EqualFold(g.Name, name) {
                group = g
                break
            }
        }
        if group == nil {
            resp.Diagnostics.AddError(
                "Group Not Found",
                "No Tenable VM group was found with name " + name + ".",
            )
            return
        }
    } else {
        resp.Diagnostics.AddError(
            "Missing Search Parameter",
            "Either the id or name attribute must be set to look up a Tenable VM group.",
        )
        return
    }
    var state groupDataSourceModel
    state.ID = types.StringValue(strconv.Itoa(group.ID))
    state.Name = types.StringValue(group.Name)
    state.UUID = types.StringValue(group.UUID)
    if group.Description != "" {
        state.Description = types.StringValue(group.Description)
    } else {
        state.Description = types.StringNull()
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) 
    // Log info message
    tflog.Info(ctx, "Read Tenable VM group data source", map[string]any{
        "group_id": state.ID.ValueString(),
        "name":     state.Name.ValueString(),
    })
}
