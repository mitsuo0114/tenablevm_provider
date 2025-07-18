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

func buildConfig(ctx context.Context, sch schema.Schema, attrs map[string]tftypes.Value) tfsdk.Config {
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

func emptyState(ctx context.Context, sch schema.Schema) tfsdk.State {
	return tfsdk.State{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(ctx), nil)}
}

func TestGroupDataSourceReadByID(t *testing.T) {
	ctx := context.Background()

	sample := []map[string]interface{}{
		{"id": 10, "uuid": "group-uuid1", "name": "Developers", "description": "Dev group"},
		{"id": 20, "uuid": "group-uuid2", "name": "Admins", "description": "Admin group"},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/groups" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sample)
	}))
	defer ts.Close()

	ds := &groupDataSource{client: newTestClient(ts)}
	var schResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schResp)

	idVal, _ := types.StringValue("10").ToTerraformValue(ctx)
	req := datasource.ReadRequest{Config: buildConfig(ctx, schResp.Schema, map[string]tftypes.Value{"id": idVal})}
	resp := datasource.ReadResponse{State: emptyState(ctx, schResp.Schema)}

	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var state groupDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("state decode error: %v", diags)
	}
	if state.ID.ValueString() != "10" || state.Name.ValueString() != "Developers" ||
		state.UUID.ValueString() != "group-uuid1" || state.Description.ValueString() != "Dev group" {
		t.Errorf("unexpected state: %+v", state)
	}
}

func TestGroupDataSourceReadByName(t *testing.T) {
	ctx := context.Background()

	sample := []map[string]interface{}{
		{"id": 10, "uuid": "group-uuid1", "name": "Developers"},
		{"id": 20, "uuid": "group-uuid2", "name": "Admins"},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/groups" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sample)
	}))
	defer ts.Close()

	ds := &groupDataSource{client: newTestClient(ts)}
	var schResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schResp)

	nameVal, _ := types.StringValue("Admins").ToTerraformValue(ctx)
	req := datasource.ReadRequest{Config: buildConfig(ctx, schResp.Schema, map[string]tftypes.Value{"name": nameVal})}
	resp := datasource.ReadResponse{State: emptyState(ctx, schResp.Schema)}

	ds.Read(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	var state groupDataSourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("state decode error: %v", diags)
	}
	if state.ID.ValueString() != "20" || state.Name.ValueString() != "Admins" {
		t.Errorf("unexpected state: %+v", state)
	}
}
