package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

// TestNewProvider_Metadata verifies that Metadata returns the expected
// type name and version string.
func TestNewProvider_Metadata(t *testing.T) {
	p := NewProvider("1.2.3").(*tenablevmProvider)
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)

	if resp.TypeName != "tenablevm" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "tenablevm")
	}
	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", resp.Version, "1.2.3")
	}
}

// TestProvider_Schema verifies that Schema defines the expected provider
// configuration attributes.
func TestProvider_Schema(t *testing.T) {
	p := NewProvider("test").(*tenablevmProvider)
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	s := resp.Schema
	attr, ok := s.Attributes["access_key"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("access_key attribute missing or wrong type")
	}
	if !attr.Optional {
		t.Errorf("access_key Optional = %v, want true", attr.Optional)
	}
	if attr.Sensitive {
		t.Errorf("access_key Sensitive = %v, want false", attr.Sensitive)
	}

	attr, ok = s.Attributes["secret_key"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("secret_key attribute missing or wrong type")
	}
	if !attr.Optional {
		t.Errorf("secret_key Optional = %v, want true", attr.Optional)
	}
	if !attr.Sensitive {
		t.Errorf("secret_key Sensitive = %v, want true", attr.Sensitive)
	}
}

// TestProvider_Resources verifies that the provider exposes the expected
// resource implementations.
func TestProvider_Resources(t *testing.T) {
	p := NewProvider("test").(*tenablevmProvider)
	rs := p.Resources(context.Background())
	if len(rs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(rs))
	}
	r := rs[0]()
	if _, ok := r.(*userResource); !ok {
		t.Fatalf("first resource type = %T, want *userResource", r)
	}
}

// TestProvider_DataSources verifies that the provider exposes the expected
// data source implementations.
func TestProvider_DataSources(t *testing.T) {
	p := NewProvider("test").(*tenablevmProvider)
	ds := p.DataSources(context.Background())
	if len(ds) != 3 {
		t.Fatalf("expected 3 data sources, got %d", len(ds))
	}
	if _, ok := ds[0]().(*userDataSource); !ok {
		t.Errorf("first data source = %T, want *userDataSource", ds[0]())
	}
	if _, ok := ds[1]().(*roleDataSource); !ok {
		t.Errorf("second data source = %T, want *roleDataSource", ds[1]())
	}
	if _, ok := ds[2]().(*groupDataSource); !ok {
		t.Errorf("third data source = %T, want *groupDataSource", ds[2]())
	}
}
