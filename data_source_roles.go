package main

import (
    "context"
    "strconv"

    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// rolesDataSource implements a data source that lists all Tenable VM
// roles.  Roles define sets of privileges that can be assigned to
// users or groups.  The underlying API returns an array of role
// objects; each role may include an ID, UUID, name and description.
// The pyTenable documentation notes that the list method returns "the
// list of roles objects"【730874566695972†L238-L245】, which we
// leverage here.
type rolesDataSource struct {
    client *Client
}

// rolesDataSourceModel defines the state structure for the roles
// data source.  Each role is represented by a nested object with
// string fields for id, uuid, name and description.
type rolesDataSourceModel struct {
    Roles []roleModel `tfsdk:"roles"`
}

// roleModel maps a single role into Terraform state.
type roleModel struct {
    ID          types.String `tfsdk:"id"`
    UUID        types.String `tfsdk:"uuid"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
}

// NewRolesDataSource returns a new roles data source.  The provider
// calls this function when registering data sources.
func NewRolesDataSource() datasource.DataSource {
    return &rolesDataSource{}
}

// Metadata sets the data source type name.  Using a plural name
// clarifies that this data source returns a list of roles.
func (d *rolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_roles"
}

// Schema defines the output structure of the roles data source.  A
// single attribute `roles` contains a list of role objects with
// computed fields.  No input attributes are required.
func (d *rolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "roles": schema.ListNestedAttribute{
                Computed: true,
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "id": schema.StringAttribute{
                            Computed:    true,
                            Description: "Numeric identifier of the role.",
                        },
                        "uuid": schema.StringAttribute{
                            Computed:    true,
                            Description: "UUID of the role.",
                        },
                        "name": schema.StringAttribute{
                            Computed:    true,
                            Description: "Name of the role.",
                        },
                        "description": schema.StringAttribute{
                            Computed:    true,
                            Description: "Description of the role.",
                        },
                    },
                },
                Description:         "List of roles available in Tenable VM.",
                MarkdownDescription: "List of roles available in Tenable VM.",
            },
        },
        Description:         "Retrieves all roles from Tenable Vulnerability Management.",
        MarkdownDescription: "Retrieves all roles from Tenable Vulnerability Management.",
    }
}

// Configure stores the API client on the data source.  The framework
// ensures this is called before Read.  If no provider data is
// provided, the client will remain nil and Read will no-op.
func (d *rolesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }
    c, ok := req.ProviderData.(*Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Provider Data Type",
            "The provider data supplied to the tenablevm_roles data source is not a *Client. This is a bug in the provider implementation.",
        )
        return
    }
    d.client = c
}

// Read invokes the client's ListRoles method and constructs the
// Terraform state.  Each role returned by the API is mapped into a
// nested object.  Errors are appended to diagnostics.
func (d *rolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    // If the client is nil, silently return.  This occurs in test
    // scenarios where provider configuration is intentionally
    // omitted.
    if d.client == nil {
        return
    }
    roles, err := d.client.ListRoles()
    if err != nil {
        resp.Diagnostics.AddError(
            "Error listing Tenable VM roles",
            err.Error(),
        )
        return
    }
    // Populate the data source model with roles
    var state rolesDataSourceModel
    for _, r := range roles {
        rm := roleModel{
            ID:   types.StringValue(strconv.Itoa(r.ID)),
            UUID: types.StringValue(r.UUID),
            Name: types.StringValue(r.Name),
        }
        if r.Description != "" {
            rm.Description = types.StringValue(r.Description)
        } else {
            rm.Description = types.StringNull()
        }
        state.Roles = append(state.Roles, rm)
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}