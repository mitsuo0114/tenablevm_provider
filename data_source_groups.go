package main

import (
    "context"
    "strconv"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// groupsDataSource implements a data source that lists all Tenable VM
// user groups.  Groups are used to manage collections of users and
// their access.  The underlying API returns an array of group
// objects; each group may include an ID, UUID, name and description.
// The pyTenable documentation for groups.list() describes it as
// returning a list of group resource records【308594680530685†L327-L334】.
type groupsDataSource struct {
    client *Client
}

// groupsDataSourceModel defines the state structure for the groups
// data source.  Each group is represented by a nested object with
// string fields for id, uuid, name and description.
type groupsDataSourceModel struct {
    Groups []groupModel `tfsdk:"groups"`
}

// groupModel maps a single group into Terraform state.
type groupModel struct {
    ID          types.String `tfsdk:"id"`
    UUID        types.String `tfsdk:"uuid"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
}

// NewGroupsDataSource returns a new groups data source.
func NewGroupsDataSource() datasource.DataSource {
    return &groupsDataSource{}
}

// Metadata sets the data source type name.  Using a plural name
// clarifies that this data source returns a list of groups.
func (d *groupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_groups"
}

// Schema defines the output structure of the groups data source.  A
// single attribute `groups` contains a list of group objects with
// computed fields.  No input attributes are required.
func (d *groupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "groups": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                            Computed:    true,
                            Description: "Numeric identifier of the group.",
                        },
                        "uuid": schema.StringAttribute{
                            Computed:    true,
                            Description: "UUID of the group.",
                        },
                        "name": schema.StringAttribute{
                            Computed:    true,
                            Description: "Name of the group.",
                        },
                        "description": schema.StringAttribute{
                            Computed:    true,
                            Description: "Description of the group.",
                        },
                    },
                },
                Description:         "List of user groups available in Tenable VM.",
                MarkdownDescription: "List of user groups available in Tenable VM.",
            },
        },
        Description:         "Retrieves all user groups from Tenable Vulnerability Management.",
        MarkdownDescription: "Retrieves all user groups from Tenable Vulnerability Management.",
    }
}

// Configure stores the API client on the data source.
func (d *groupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Provider Data Type",
            "The provider data supplied to the tenablevm_groups data source is not a *Client. This is a bug in the provider implementation.",
        )
        return
    }
    d.client = c
}

// Read invokes the client's ListGroups method and constructs the
// Terraform state.  Each group returned by the API is mapped into a
// nested object.  Errors are appended to diagnostics.
func (d *groupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // If the client is nil, do nothing.  This occurs during tests.
    if d.client == nil {
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
    // Populate the data source model with groups
    var state groupsDataSourceModel
    for _, g := range groups {
        gm := groupModel{
            ID:   types.StringValue(strconv.Itoa(g.ID)),
            UUID: types.StringValue(g.UUID),
            Name: types.StringValue(g.Name),
        }
        if g.Description != "" {
            gm.Description = types.StringValue(g.Description)
        } else {
            gm.Description = types.StringNull()
        }
        state.Groups = append(state.Groups, gm)
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}