package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

// Client encapsulates low‑level interactions with the Tenable
// Vulnerability Management REST API.  It handles HTTP request
// construction, authentication header insertion, and response
// decoding.  Each method returns a parsed response or an error.
//
// This implementation is shared between both the legacy Terraform
// provider and the modern plugin‑framework based provider.  See
// provider.go for the provider integration.
const baseURL = "https://cloud.tenable.com"

type Client struct {
    AccessKey string
    SecretKey string
    Http      *http.Client
}

// newRequest constructs an HTTP request for the given path and
// optional JSON body.  The path is appended to the base URL and
// authentication headers are applied.  The caller is responsible for
// executing the returned request.
func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
    url := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")

    var buf io.Reader
    if body != nil {
        b := new(bytes.Buffer)
        if err := json.NewEncoder(b).Encode(body); err != nil {
            return nil, err
        }
        buf = b
    }

    req, err := http.NewRequest(method, url, buf)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    // According to Tenable's API documentation, clients must set the
    // X-ApiKeys header using the access key and secret key for
    // authentication【507416795845449†L142-L160】.
    req.Header.Set("X-ApiKeys", fmt.Sprintf("accessKey=%s; secretKey=%s;", c.AccessKey, c.SecretKey))
    return req, nil
}

// do executes the HTTP request and decodes the JSON response into
// target if provided.  Non‑2xx responses result in an error with the
// body text included for debugging.  A nil target suppresses decoding
// entirely.
func (c *Client) do(req *http.Request, target interface{}) error {
    resp, err := c.Http.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        // read body for error message
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("API error: %s: %s", resp.Status, string(bodyBytes))
    }
    if target == nil {
        return nil
    }
    return json.NewDecoder(resp.Body).Decode(target)
}

// User represents a Tenable VM user resource.  Only a subset of
// fields are defined here; additional fields returned by the API
// will be captured in the Raw map.
type User struct {
    ID          int                    `json:"id"`
    UUID        string                 `json:"uuid"`
    Username    string                 `json:"username"`
    Name        string                 `json:"name"`
    Email       string                 `json:"email"`
    Permissions int                    `json:"permissions"`
    Enabled     bool                   `json:"enabled"`
    Raw         map[string]interface{} `json:"-"`
}

// Role represents a Tenable VM role (custom role).  Only a subset
// of fields are defined here; additional fields returned by the API
// are captured in Raw.  Roles define a set of privileges and can be
// assigned to users or groups.
type Role struct {
    ID          int                    `json:"id"`
    UUID        string                 `json:"uuid"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Raw         map[string]interface{} `json:"-"`
}

// Group represents a Tenable VM user group.  Groups are used to
// manage collections of users and their access.  Only common fields
// are explicitly defined; other fields are stored in Raw.
type Group struct {
    ID          int                    `json:"id"`
    UUID        string                 `json:"uuid"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Raw         map[string]interface{} `json:"-"`
}

// CreateUser creates a new user in Tenable VM.  The returned user
// structure includes the generated user ID which is used to set the
// Terraform resource ID.  See Tenable's API documentation for
// supported permissions values【946957473917885†L60-L74】.
func (c *Client) CreateUser(username, password string, permissions int, name, email, accountType string, enabled bool) (*User, error) {
    payload := map[string]interface{}{
        "username":    username,
        "password":    password,
        "permissions": permissions,
        "type":        accountType,
    }
    if name != "" {
        payload["name"] = name
    }
    if email != "" {
        payload["email"] = email
    }
    // Issue the create request
    req, err := c.newRequest(http.MethodPost, "users", payload)
    if err != nil {
        return nil, err
    }
    var resp map[string]interface{}
    if err := c.do(req, &resp); err != nil {
        return nil, err
    }
    // The API returns the created user record.  Extract the ID and
    // enabled state.  Some Tenable deployments may not include an
    // explicit 'enabled' field on creation, so default to true.
    user := &User{Raw: resp}
    if v, ok := resp["id"]; ok {
        switch id := v.(type) {
        case float64:
            user.ID = int(id)
        case int:
            user.ID = id
        }
    }
    if v, ok := resp["uuid"]; ok {
        if s, ok := v.(string); ok {
            user.UUID = s
        }
    }
    if v, ok := resp["username"]; ok {
        if s, ok := v.(string); ok {
            user.Username = s
        }
    }
    if v, ok := resp["name"]; ok {
        if s, ok := v.(string); ok {
            user.Name = s
        }
    }
    if v, ok := resp["email"]; ok {
        if s, ok := v.(string); ok {
            user.Email = s
        }
    }
    if v, ok := resp["permissions"]; ok {
        switch p := v.(type) {
        case float64:
            user.Permissions = int(p)
        case int:
            user.Permissions = p
        }
    }
    if v, ok := resp["enabled"]; ok {
        if b, ok := v.(bool); ok {
            user.Enabled = b
        }
    } else {
        user.Enabled = true
    }
    // If the enabled flag in the payload differs from the API
    // response, update it accordingly using the dedicated endpoint.
    if user.ID != 0 && user.Enabled != enabled {
        if err := c.SetUserEnabled(user.ID, enabled); err != nil {
            return nil, err
        }
        user.Enabled = enabled
    }
    return user, nil
}

// GetUser retrieves the details of a user by ID【946957473917885†L95-L113】.
func (c *Client) GetUser(id int) (*User, error) {
    req, err := c.newRequest(http.MethodGet, fmt.Sprintf("users/%d", id), nil)
    if err != nil {
        return nil, err
    }
    var resp map[string]interface{}
    if err := c.do(req, &resp); err != nil {
        return nil, err
    }
    user := &User{Raw: resp}
    if v, ok := resp["id"]; ok {
        switch idv := v.(type) {
        case float64:
            user.ID = int(idv)
        case int:
            user.ID = idv
        }
    }
    if v, ok := resp["uuid"]; ok {
        user.UUID, _ = v.(string)
    }
    if v, ok := resp["username"]; ok {
        user.Username, _ = v.(string)
    }
    if v, ok := resp["name"]; ok {
        user.Name, _ = v.(string)
    }
    if v, ok := resp["email"]; ok {
        user.Email, _ = v.(string)
    }
    if v, ok := resp["permissions"]; ok {
        switch p := v.(type) {
        case float64:
            user.Permissions = int(p)
        case int:
            user.Permissions = p
        }
    }
    if v, ok := resp["enabled"]; ok {
        if b, ok := v.(bool); ok {
            user.Enabled = b
        }
    }
    return user, nil
}

// ListUsers retrieves all users from Tenable VM.  The returned slice
// contains basic information for each user.  This method is used by
// data sources to locate a user by username when only the username
// is known.  The API returns a list of user objects; each user
// record may include only a subset of fields depending on the
// requesting user's permissions【515179993953485†L793-L802】.
func (c *Client) ListUsers() ([]*User, error) {
    req, err := c.newRequest(http.MethodGet, "users", nil)
    if err != nil {
        return nil, err
    }
    // Decode the response into a slice of maps.  According to Tenable's
    // API documentation, the list endpoint returns a JSON array of
    // user objects【515179993953485†L793-L802】.  Each object may contain
    // fields such as id, uuid, username, name, email, permissions and
    // enabled, though not all fields are guaranteed to be present.
    var resp []map[string]interface{}
    if err := c.do(req, &resp); err != nil {
        return nil, err
    }
    users := make([]*User, 0, len(resp))
    for _, m := range resp {
        user := &User{Raw: m}
        if v, ok := m["id"]; ok {
            switch id := v.(type) {
            case float64:
                user.ID = int(id)
            case int:
                user.ID = id
            }
        }
        if v, ok := m["uuid"]; ok {
            if s, ok := v.(string); ok {
                user.UUID = s
            }
        }
        if v, ok := m["username"]; ok {
            if s, ok := v.(string); ok {
                user.Username = s
            }
        }
        if v, ok := m["name"]; ok {
            if s, ok := v.(string); ok {
                user.Name = s
            }
        }
        if v, ok := m["email"]; ok {
            if s, ok := v.(string); ok {
                user.Email = s
            }
        }
        if v, ok := m["permissions"]; ok {
            switch p := v.(type) {
            case float64:
                user.Permissions = int(p)
            case int:
                user.Permissions = p
            }
        }
        if v, ok := m["enabled"]; ok {
            if b, ok := v.(bool); ok {
                user.Enabled = b
            }
        }
        users = append(users, user)
    }
    return users, nil
}

// ListRoles retrieves all roles from Tenable VM.  The roles API
// returns an array of role objects representing custom roles.  Each
// object may include fields such as id, uuid, name, and description.
// See the pyTenable documentation which notes that list() returns
// "the list of roles objects"【730874566695972†L238-L245】.
func (c *Client) ListRoles() ([]*Role, error) {
    req, err := c.newRequest(http.MethodGet, "roles", nil)
    if err != nil {
        return nil, err
    }
    var resp []map[string]interface{}
    if err := c.do(req, &resp); err != nil {
        return nil, err
    }
    roles := make([]*Role, 0, len(resp))
    for _, m := range resp {
        role := &Role{Raw: m}
        if v, ok := m["id"]; ok {
            switch id := v.(type) {
            case float64:
                role.ID = int(id)
            case int:
                role.ID = id
            }
        }
        if v, ok := m["uuid"]; ok {
            if s, ok := v.(string); ok {
                role.UUID = s
            }
        }
        if v, ok := m["name"]; ok {
            if s, ok := v.(string); ok {
                role.Name = s
            }
        }
        if v, ok := m["description"]; ok {
            if s, ok := v.(string); ok {
                role.Description = s
            }
        }
        roles = append(roles, role)
    }
    return roles, nil
}

// ListGroups retrieves all user groups from Tenable VM.  The groups
// API returns an array of group objects.  The pyTenable
// documentation for groups.list() states that it "lists all of the
// available user groups" and returns a list of group resource
// records【308594680530685†L327-L334】.  Each group may include id,
// uuid, name and description fields.
func (c *Client) ListGroups() ([]*Group, error) {
    req, err := c.newRequest(http.MethodGet, "groups", nil)
    if err != nil {
        return nil, err
    }
    var resp []map[string]interface{}
    if err := c.do(req, &resp); err != nil {
        return nil, err
    }
    groups := make([]*Group, 0, len(resp))
    for _, m := range resp {
        group := &Group{Raw: m}
        if v, ok := m["id"]; ok {
            switch id := v.(type) {
            case float64:
                group.ID = int(id)
            case int:
                group.ID = id
            }
        }
        if v, ok := m["uuid"]; ok {
            if s, ok := v.(string); ok {
                group.UUID = s
            }
        }
        if v, ok := m["name"]; ok {
            if s, ok := v.(string); ok {
                group.Name = s
            }
        }
        if v, ok := m["description"]; ok {
            if s, ok := v.(string); ok {
                group.Description = s
            }
        }
        groups = append(groups, group)
    }
    return groups, nil
}

// UpdateUser modifies an existing user.  Only non‑zero/non‑empty
// attributes are applied.  Permissions and enabled state are
// optional.  The Tenable API requires a PUT request to
// /users/{id} to update name, email, permissions and enabled
// properties as described in the pyTenable implementation【946957473917885†L143-L165】.
func (c *Client) UpdateUser(id int, permissions *int, name, email *string, enabled *bool) (*User, error) {
    // Build payload by merging existing values with desired
    current, err := c.GetUser(id)
    if err != nil {
        return nil, err
    }
    payload := map[string]interface{}{}
    // Always send current permissions, enabled, email, name; then override
    payload["permissions"] = current.Permissions
    payload["enabled"] = current.Enabled
    payload["email"] = current.Email
    payload["name"] = current.Name
    if permissions != nil {
        payload["permissions"] = *permissions
    }
    if enabled != nil {
        payload["enabled"] = *enabled
    }
    if email != nil {
        payload["email"] = *email
    }
    if name != nil {
        payload["name"] = *name
    }
    req, err := c.newRequest(http.MethodPut, fmt.Sprintf("users/%d", id), payload)
    if err != nil {
        return nil, err
    }
    var resp map[string]interface{}
    if err := c.do(req, &resp); err != nil {
        return nil, err
    }
    // update and return user
    return c.GetUser(id)
}

// DeleteUser removes a user from Tenable VM【946957473917885†L76-L93】.
func (c *Client) DeleteUser(id int) error {
    req, err := c.newRequest(http.MethodDelete, fmt.Sprintf("users/%d", id), nil)
    if err != nil {
        return err
    }
    // Tenable's delete endpoint returns empty body on success
    return c.do(req, nil)
}

// SetUserEnabled toggles a user's enabled status using the dedicated
// endpoint.  This helper is used after creation to ensure the
// resource reflects the desired enabled flag【946957473917885†L167-L193】.
func (c *Client) SetUserEnabled(id int, enabled bool) error {
    payload := map[string]interface{}{
        "enabled": enabled,
    }
    req, err := c.newRequest(http.MethodPut, fmt.Sprintf("users/%d/enabled", id), payload)
    if err != nil {
        return err
    }
    return c.do(req, nil)
}