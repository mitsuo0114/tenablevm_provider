package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "net/url"
    "reflect"
    "testing"
)

type rewriteTransport struct {
    base *url.URL
    rt   http.RoundTripper
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    u := *req.URL
    u.Scheme = r.base.Scheme
    u.Host = r.base.Host
    req.URL = &u
    return r.rt.RoundTrip(req)
}

func newTestClient(ts *httptest.Server) *Client {
    base, _ := url.Parse(ts.URL)
    return &Client{
        AccessKey: "access",
        SecretKey: "secret",
        Http: &http.Client{Transport: rewriteTransport{base: base, rt: ts.Client().Transport}},
    }
}

// TestClient_newRequestHeaders verifies that newRequest sets the X-ApiKeys header
// and Content-Type for JSON bodies.  This ensures API authentication headers
// conform to Tenable's specification.
func TestClient_newRequestHeaders(t *testing.T) {
    client := &Client{
        AccessKey: "access123",
        SecretKey: "secret456",
        Http:      http.DefaultClient,
    }
    req, err := client.newRequest(http.MethodGet, "users", nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got, want := req.Header.Get("X-ApiKeys"), "accessKey=access123; secretKey=secret456;"; got != want {
        t.Errorf("X-ApiKeys header = %q, want %q", got, want)
    }
    if got, want := req.Header.Get("Content-Type"), "application/json"; got != want {
        t.Errorf("Content-Type header = %q, want %q", got, want)
    }
}

// TestClient_ListUsers verifies that ListUsers parses a list of users
// correctly from the API and returns the expected slice of User structs.
func TestClient_ListUsers(t *testing.T) {
    // Sample JSON response representing two users
    sample := []map[string]interface{}{
        {
            "id":         1,
            "uuid":       "uuid-1",
            "username":   "alice",
            "name":       "Alice",
            "email":      "alice@example.com",
            "permissions": 16,
            "enabled":    true,
        },
        {
            "id":         2,
            "uuid":       "uuid-2",
            "username":   "bob",
            "name":       "Bob",
            "email":      "bob@example.com",
            "permissions": 32,
            "enabled":    false,
        },
    }
    // Create a test server that returns the sample response
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/users" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(sample)
    }))
    defer ts.Close()
    client := newTestClient(ts)
    users, err := client.ListUsers()
    if err != nil {
        t.Fatalf("ListUsers error: %v", err)
    }
    if len(users) != len(sample) {
        t.Fatalf("got %d users, want %d", len(users), len(sample))
    }
    // Compare each user
    for i, u := range users {
        if u.ID != sample[i]["id"].(int) {
            t.Errorf("user %d ID mismatch: got %d, want %d", i, u.ID, sample[i]["id"].(int))
        }
        // We'll compare all fields manually using reflect.DeepEqual on a map
        expected := &User{
            ID:          int(sample[i]["id"].(int)),
            UUID:        sample[i]["uuid"].(string),
            Username:    sample[i]["username"].(string),
            Name:        sample[i]["name"].(string),
            Email:       sample[i]["email"].(string),
            Permissions: int(sample[i]["permissions"].(int)),
            Enabled:     sample[i]["enabled"].(bool),
        }
        if !reflect.DeepEqual(u.ID, expected.ID) || u.UUID != expected.UUID || u.Username != expected.Username || u.Name != expected.Name || u.Email != expected.Email || u.Permissions != expected.Permissions || u.Enabled != expected.Enabled {
            t.Errorf("user %d mismatch\n got: %+v\nwant: %+v", i, u, expected)
        }
    }
}

// TestClient_GetUser verifies that GetUser retrieves and parses a single user.
func TestClient_GetUser(t *testing.T) {
    sample := map[string]interface{}{
        "id":         1,
        "uuid":       "uuid-1",
        "username":   "alice",
        "name":       "Alice",
        "email":      "alice@example.com",
        "permissions": 16,
        "enabled":    true,
    }
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/users/1" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(sample)
    }))
    defer ts.Close()
    client := newTestClient(ts)
    user, err := client.GetUser(1)
    if err != nil {
        t.Fatalf("GetUser error: %v", err)
    }
    expected := &User{
        ID:          int(sample["id"].(int)),
        UUID:        sample["uuid"].(string),
        Username:    sample["username"].(string),
        Name:        sample["name"].(string),
        Email:       sample["email"].(string),
        Permissions: int(sample["permissions"].(int)),
        Enabled:     sample["enabled"].(bool),
    }
    if !reflect.DeepEqual(user, expected) {
        t.Errorf("GetUser mismatch\n got: %+v\nwant: %+v", user, expected)
    }
}

// TestClient_ListRoles verifies that ListRoles parses role arrays correctly.
func TestClient_ListRoles(t *testing.T) {
    sample := []map[string]interface{}{
        {
            "id":          1,
            "uuid":        "role-uuid1",
            "name":        "Reader",
            "description": "Read only access",
        },
        {
            "id":          2,
            "uuid":        "role-uuid2",
            "name":        "Admin",
            "description": "Admin access",
        },
    }
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/roles" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(sample)
    }))
    defer ts.Close()
    client := newTestClient(ts)
    roles, err := client.ListRoles()
    if err != nil {
        t.Fatalf("ListRoles error: %v", err)
    }
    if len(roles) != len(sample) {
        t.Fatalf("got %d roles, want %d", len(roles), len(sample))
    }
    for i, r := range roles {
        expected := &Role{
            ID:          int(sample[i]["id"].(int)),
            UUID:        sample[i]["uuid"].(string),
            Name:        sample[i]["name"].(string),
            Description: sample[i]["description"].(string),
        }
        if !reflect.DeepEqual(r, expected) {
            t.Errorf("role %d mismatch\n got: %+v\nwant: %+v", i, r, expected)
        }
    }
}

// TestClient_ListGroups verifies that ListGroups parses group arrays correctly.
func TestClient_ListGroups(t *testing.T) {
    sample := []map[string]interface{}{
        {
            "id":          10,
            "uuid":        "group-uuid1",
            "name":        "Developers",
            "description": "Dev group",
        },
        {
            "id":          20,
            "uuid":        "group-uuid2",
            "name":        "Admins",
            "description": "Admin group",
        },
    }
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/groups" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(sample)
    }))
    defer ts.Close()
    client := newTestClient(ts)
    groups, err := client.ListGroups()
    if err != nil {
        t.Fatalf("ListGroups error: %v", err)
    }
    if len(groups) != len(sample) {
        t.Fatalf("got %d groups, want %d", len(groups), len(sample))
    }
    for i, g := range groups {
        expected := &Group{
            ID:          int(sample[i]["id"].(int)),
            UUID:        sample[i]["uuid"].(string),
            Name:        sample[i]["name"].(string),
            Description: sample[i]["description"].(string),
        }
        if !reflect.DeepEqual(g, expected) {
            t.Errorf("group %d mismatch\n got: %+v\nwant: %+v", i, g, expected)
        }
    }
}
