package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func buildUserConfig(ctx context.Context, sch schema.Schema, attrs map[string]tftypes.Value) tfsdk.Config {
	attrTypes := make(map[string]tftypes.Type)
	vals := make(map[string]tftypes.Value)
	for name, attr := range sch.Attributes {
		typ := attr.GetType().TerraformType(ctx)
		attrTypes[name] = typ
		if v, ok := attrs[name]; ok {
			vals[name] = v
		} else {
			vals[name] = tftypes.NewValue(typ, nil)
		}
	}
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, vals)
	return tfsdk.Config{Schema: sch, Raw: raw}
}

func TestUserDataSourceReadByID(t *testing.T) {
	ctx := context.Background()

	sample := map[string]interface{}{
		"id": 1, "uuid": "uuid-1", "username": "alice", "name": "Alice",
		"email": "alice@example.com", "permissions": 16, "enabled": true,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sample)
	}))
	defer ts.Close()

	ds := &userDataSource{client: newTestClient(ts)}

	var schResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schResp)

	idVal, _ := types.StringValue("1").ToTerraformValue(ctx)
	req := datasource.ReadRequest{Config: buildUserConfig(ctx, schResp.Schema, map[string]tftypes.Value{"id": idVal})}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schResp.Schema, Raw: tftypes.NewValue(schResp.Schema.Type().TerraformType(ctx), nil)}}

	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var state userDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("state decode error: %v", diags)
	}
	if state.ID.ValueString() != "1" || state.Username.ValueString() != "alice" ||
		state.Email.ValueString() != "alice@example.com" || !state.Enabled.ValueBool() {
		t.Errorf("unexpected state: %+v", state)
	}
}

func TestUserDataSourceReadByUsername(t *testing.T) {
	ctx := context.Background()

	list := []map[string]interface{}{
		{"id": 1, "uuid": "uuid-1", "username": "alice"},
		{"id": 2, "uuid": "uuid-2", "username": "bob"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		case "/users/2":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": 2, "uuid": "uuid-2", "username": "bob", "permissions": 16, "enabled": true})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	ds := &userDataSource{client: newTestClient(ts)}

	var schResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schResp)

	userVal, _ := types.StringValue("bob").ToTerraformValue(ctx)
	req := datasource.ReadRequest{Config: buildUserConfig(ctx, schResp.Schema, map[string]tftypes.Value{"username": userVal})}
	resp := datasource.ReadResponse{State: tfsdk.State{Schema: schResp.Schema, Raw: tftypes.NewValue(schResp.Schema.Type().TerraformType(ctx), nil)}}

	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var state userDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("state decode error: %v", diags)
	}
	if state.ID.ValueString() != "2" || state.Username.ValueString() != "bob" {
		t.Errorf("unexpected state: %+v", state)
	}
}
