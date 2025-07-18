package main

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	// Structured logging for resources
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource implementation satisfies the expected interfaces.
var _ resource.Resource = &userResource{}
var _ resource.ResourceWithConfigure = &userResource{}
var _ resource.ResourceWithImportState = &userResource{}

// userResource implements the Terraform resource for managing Tenable VM
// users.  It embeds a client pointer which is configured by the
// provider.  Each CRUD method uses the client to interact with
// Tenable's API.
type userResource struct {
	client *Client
}

// NewUserResource returns a new instance of the user resource.  This
// function is used by the provider to instantiate the resource.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResourceModel maps the resource schema data into a Go struct.  The
// `tfsdk` tags correspond to the schema attribute names.  All
// attributes leverage the framework's types to track null/unknown
// values.
type userResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Permissions types.Int64  `tfsdk:"permissions"`
	Name        types.String `tfsdk:"name"`
	Email       types.String `tfsdk:"email"`
	AccountType types.String `tfsdk:"account_type"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

// Metadata sets the resource type name.  The type name is appended
// onto the provider type name to form the full resource identifier
// (e.g. tenablevm_user).
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the Tenable VM user resource.  It
// closely mirrors the fields accepted by Tenable's API while
// adhering to Terraform semantics.  Certain attributes, such as
// username, password and account_type, are marked with plan
// modifiers to force a new resource if they change, since the
// underlying API does not allow in‑place modification of these
// values.  The password is write‑only and sensitive so it is never
// persisted in state.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Numeric identifier of the user.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "Numeric identifier of the user.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				Description:         "The username for the Tenable VM user. Must be unique.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				MarkdownDescription: "The username for the Tenable VM user. Must be unique.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				Description:         "Password for the user. Password updates are not supported; changing this forces replacement.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				MarkdownDescription: "Password for the user. Password updates are not supported; changing this forces replacement.",
			},
			"permissions": schema.Int64Attribute{
				Required:            true,
				Description:         "Numeric permissions role for the user. See Tenable's user roles documentation for valid values【946957473917885†L60-L74】.",
				MarkdownDescription: "Numeric permissions role for the user. See Tenable's user roles documentation for valid values【946957473917885†L60-L74】.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Description:         "Human‑readable name of the user.",
				MarkdownDescription: "Human‑readable name of the user.",
			},
			"email": schema.StringAttribute{
				Optional:            true,
				Description:         "Email address for the user.",
				MarkdownDescription: "Email address for the user.",
			},
			"account_type": schema.StringAttribute{
				Optional:            true,
				Description:         "Account type for the user (e.g. local). Changing this forces a new user to be created.",
				MarkdownDescription: "Account type for the user (e.g. local). Changing this forces a new user to be created.",
				Default:             stringdefault.StaticString("local"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Description:         "Whether the user account is enabled.",
				MarkdownDescription: "Whether the user account is enabled.",
				Default:             booldefault.StaticBool(true),
			},
		},
		Description:         "Manages a Tenable Vulnerability Management user account.",
		MarkdownDescription: "Manages a Tenable Vulnerability Management user account.",
	}
}

// Configure sets the API client on the resource.  If the provider did
// not supply client data (e.g. during unit testing), the resource
// gracefully skips configuration.  Any type mismatches result in a
// diagnostic error.
func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			"The provider data supplied to the tenablevm_user resource is not a *Client. This is a bug in the provider implementation.",
		)
		return
	}
	r.client = client
}

// Create implements the resource creation logic.  It reads the plan
// values, invokes the client's CreateUser method, and persists the
// resulting state.  Unknown or invalid plan values result in
// diagnostics.  The password is not persisted to state.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve plan into model
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Extract values from plan
	username := plan.Username.ValueString()
	password := ""
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		password = plan.Password.ValueString()
	}
	permissions := int(plan.Permissions.ValueInt64())
	var name string
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		name = plan.Name.ValueString()
	}
	var email string
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		email = plan.Email.ValueString()
	}
	accountType := "local"
	if !plan.AccountType.IsNull() && !plan.AccountType.IsUnknown() {
		accountType = plan.AccountType.ValueString()
	}
	enabled := true
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled = plan.Enabled.ValueBool()
	}
	// Log debug information about the plan before creation
	tflog.Debug(ctx, "Creating Tenable VM user", map[string]any{
		"username":    username,
		"permissions": permissions,
		"accountType": accountType,
		"enabled":     enabled,
	})

	// Call API to create user
	user, err := r.client.CreateUser(username, password, permissions, name, email, accountType, enabled)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Tenable VM user",
			err.Error(),
		)
		return
	}
	// Log info with created user ID
	tflog.Info(ctx, "Created Tenable VM user", map[string]any{
		"user_id":  user.ID,
		"username": user.Username,
	})

	// Build state from API response and plan
	var state userResourceModel
	state.ID = types.StringValue(strconv.Itoa(user.ID))
	state.Username = types.StringValue(user.Username)
	// Never persist password in state; mark as null
	state.Password = types.StringNull()
	state.Permissions = types.Int64Value(int64(user.Permissions))
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
	// AccountType is not returned by the API; use plan value
	if !plan.AccountType.IsNull() && !plan.AccountType.IsUnknown() {
		state.AccountType = types.StringValue(plan.AccountType.ValueString())
	} else {
		state.AccountType = types.StringValue(accountType)
	}
	state.Enabled = types.BoolValue(user.Enabled)
	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the resource state from the API.  If the user no
// longer exists, the state is removed.  Otherwise the latest values
// are loaded into state.  Optional attributes not returned by the
// API retain their previous values.  The password is always null in
// state.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Log debug message indicating read operation
	tflog.Debug(ctx, "Reading Tenable VM user state")

	// Get current state
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Parse ID
	idStr := state.ID.ValueString()
	id, err := strconv.Atoi(idStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid User ID",
			"Expected numeric ID but got: "+idStr,
		)
		return
	}
	// Call API to get user
	user, err := r.client.GetUser(id)
	if err != nil {
		// If the user cannot be found (e.g. 404), remove from state
		// Note: The client does not differentiate error types, so
		// remove state for any API error to ensure recreation on next
		// apply
		tflog.Info(ctx, "Tenable VM user not found during read", map[string]any{
			"user_id": state.ID.ValueString(),
			"error":   err.Error(),
		})
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddWarning(
			"Tenable VM user not found",
			"Removing tenablevm_user resource with ID "+state.ID.ValueString()+" from state due to read error: "+err.Error(),
		)
		return
	}
	// Update state with retrieved values
	state.Username = types.StringValue(user.Username)
	state.Permissions = types.Int64Value(int64(user.Permissions))
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
	// Preserve account_type from existing state since API doesn't return it
	// Preserve password as null
	state.Password = types.StringNull()
	state.Enabled = types.BoolValue(user.Enabled)
	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	// Log debug message after successful read
	tflog.Debug(ctx, "Read Tenable VM user", map[string]any{
		"user_id":  state.ID.ValueString(),
		"username": state.Username.ValueString(),
	})
}

// Update applies changes from the plan to the existing resource.  Only
// permissions, name, email and enabled can be updated.  If no
// changes are detected, the method returns without calling the API.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read plan and state
	var plan userResourceModel
	var state userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid User ID",
			"Expected numeric ID but got: "+state.ID.ValueString(),
		)
		return
	}
	// Determine which fields changed
	var perms *int
	var name *string
	var email *string
	var enabled *bool
	if plan.Permissions.ValueInt64() != state.Permissions.ValueInt64() {
		p := int(plan.Permissions.ValueInt64())
		perms = &p
	}
	// Name: If null/unknown treat as empty string
	if !plan.Name.IsUnknown() {
		// Compare plan and state values, treating null as empty
		planName := ""
		stateName := ""
		if !plan.Name.IsNull() {
			planName = plan.Name.ValueString()
		}
		if !state.Name.IsNull() {
			stateName = state.Name.ValueString()
		}
		if planName != stateName {
			s := planName
			name = &s
		}
	}
	// Email
	if !plan.Email.IsUnknown() {
		planEmail := ""
		stateEmail := ""
		if !plan.Email.IsNull() {
			planEmail = plan.Email.ValueString()
		}
		if !state.Email.IsNull() {
			stateEmail = state.Email.ValueString()
		}
		if planEmail != stateEmail {
			s := planEmail
			email = &s
		}
	}
	// Enabled
	if !plan.Enabled.IsUnknown() && plan.Enabled.ValueBool() != state.Enabled.ValueBool() {
		b := plan.Enabled.ValueBool()
		enabled = &b
	}
	// If no updatable fields changed, return early
	if perms == nil && name == nil && email == nil && enabled == nil {
		return
	}
	// Log debug message about which fields are being updated
	tflog.Debug(ctx, "Updating Tenable VM user", map[string]any{
		"user_id":             state.ID.ValueString(),
		"username":            state.Username.ValueString(),
		"permissions_changed": perms != nil,
		"name_changed":        name != nil,
		"email_changed":       email != nil,
		"enabled_changed":     enabled != nil,
	})

	// Call API to update user
	_, err = r.client.UpdateUser(id, perms, name, email, enabled)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Tenable VM user",
			err.Error(),
		)
		return
	}
	// Fetch latest user state
	updatedUser, err := r.client.GetUser(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Tenable VM user after update",
			err.Error(),
		)
		return
	}
	// Update state fields
	state.Username = types.StringValue(updatedUser.Username)
	state.Permissions = types.Int64Value(int64(updatedUser.Permissions))
	if updatedUser.Name != "" {
		state.Name = types.StringValue(updatedUser.Name)
	} else {
		state.Name = types.StringNull()
	}
	if updatedUser.Email != "" {
		state.Email = types.StringValue(updatedUser.Email)
	} else {
		state.Email = types.StringNull()
	}
	// AccountType remains unchanged
	state.Password = types.StringNull()
	state.Enabled = types.BoolValue(updatedUser.Enabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	// Log info after successful update
	tflog.Info(ctx, "Updated Tenable VM user", map[string]any{
		"user_id":  state.ID.ValueString(),
		"username": state.Username.ValueString(),
	})
}

// Delete removes the user from Tenable VM.  Any errors during
// deletion are propagated via diagnostics.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read state to get ID
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid User ID",
			"Expected numeric ID but got: "+state.ID.ValueString(),
		)
		return
	}
	// Log debug before deletion
	tflog.Debug(ctx, "Deleting Tenable VM user", map[string]any{
		"user_id":  state.ID.ValueString(),
		"username": state.Username.ValueString(),
	})
	// Call API to delete user
	if err := r.client.DeleteUser(id); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Tenable VM user",
			err.Error(),
		)
		return
	}
	// Remove resource from state
	resp.State.RemoveResource(ctx)
	// Log info after deletion
	tflog.Info(ctx, "Deleted Tenable VM user", map[string]any{
		"user_id":  state.ID.ValueString(),
		"username": state.Username.ValueString(),
	})
}

// ImportState enables users to import existing Tenable VM users into
// Terraform state.  The import ID should be the numeric user ID.
// Only the ID attribute is set; other attributes will be populated
// during the subsequent Read operation.
func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
