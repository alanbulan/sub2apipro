package feishu

import (
	"context"
	"net/http"
	"net/url"
)

var directoryScopes = []string{
	"contact:contact.base:readonly",
	"contact:user.base:readonly",
	"contact:department.base:readonly",
	"contact:user.email:readonly",
	"contact:user.employee:readonly",
}

// ConfigureDirectoryRead changes only this app's read permissions and contacts
// range. Its login app ID, callback and existing user scopes remain unchanged.
func (c *Client) ConfigureDirectoryRead(ctx context.Context, token, appID string) error {
	scopes := make([]map[string]string, 0, len(directoryScopes))
	for _, scope := range directoryScopes {
		scopes = append(scopes, map[string]string{"scope_name": scope, "token_type": "tenant"})
	}
	var out struct{}
	return c.jsonRequest(ctx, "configure_directory", http.MethodPatch, APIURL+"/application/v7/applications/"+url.PathEscape(appID)+"/config", token, map[string]any{
		"scope":    map[string]any{"add_scopes": scopes},
		"contacts": map[string]string{"contacts_range_type": "all"},
	}, &out)
}

type DirectoryMember struct {
	OpenID          string `json:"open_id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	EnterpriseEmail string `json:"enterprise_email"`
	Avatar          struct {
		URL string `json:"avatar_240"`
	} `json:"avatar"`
	Status *struct {
		Frozen    bool `json:"is_frozen"`
		Resigned  bool `json:"is_resigned"`
		Activated bool `json:"is_activated"`
		Exited    bool `json:"is_exited"`
		Unjoined  bool `json:"is_unjoin"`
	} `json:"status"`
}

func (m DirectoryMember) Active() bool {
	return m.Status != nil && m.Status.Activated && !m.Status.Frozen && !m.Status.Resigned && !m.Status.Exited && !m.Status.Unjoined
}

// DirectoryMembers obtains a complete snapshot before callers write anything.
// Both department and user pagination are mandatory; members in multiple
// departments are deduplicated by the same app-scoped open_id used at sign-in.
func (c *Client) DirectoryMembers(ctx context.Context, token string) ([]DirectoryMember, error) {
	departments := []string{"0"}
	seenDepartments := map[string]bool{"0": true}
	q := url.Values{"department_id_type": {"open_department_id"}, "user_id_type": {"open_id"}, "fetch_child": {"true"}, "page_size": {"50"}}
	seenPages := map[string]bool{}
	for {
		var out struct {
			Data *struct {
				Items []struct {
					ID string `json:"open_department_id"`
				} `json:"items"`
				HasMore bool   `json:"has_more"`
				Next    string `json:"page_token"`
			} `json:"data"`
		}
		if err := c.request(ctx, "directory_departments", http.MethodGet, APIURL+"/contact/v3/departments/0/children?"+q.Encode(), token, nil, "", &out); err != nil {
			return nil, err
		}
		if out.Data == nil {
			return nil, &Error{"directory_departments", "missing_data"}
		}
		for _, department := range out.Data.Items {
			if department.ID == "" {
				return nil, &Error{"directory_departments", "missing_department_id"}
			}
			if !seenDepartments[department.ID] {
				seenDepartments[department.ID] = true
				departments = append(departments, department.ID)
			}
		}
		if !out.Data.HasMore {
			break
		}
		if out.Data.Next == "" || seenPages[out.Data.Next] {
			return nil, &Error{"directory_departments", "invalid_pagination"}
		}
		seenPages[out.Data.Next] = true
		q.Set("page_token", out.Data.Next)
	}
	seenMembers := map[string]bool{}
	members := make([]DirectoryMember, 0)
	for _, department := range departments {
		q := url.Values{"department_id_type": {"open_department_id"}, "user_id_type": {"open_id"}, "department_id": {department}, "page_size": {"50"}}
		seenPages := map[string]bool{}
		for {
			var out struct {
				Data *struct {
					Items   []DirectoryMember `json:"items"`
					HasMore bool              `json:"has_more"`
					Next    string            `json:"page_token"`
				} `json:"data"`
			}
			if err := c.request(ctx, "directory_members", http.MethodGet, APIURL+"/contact/v3/users/find_by_department?"+q.Encode(), token, nil, "", &out); err != nil {
				return nil, err
			}
			if out.Data == nil {
				return nil, &Error{"directory_members", "missing_data"}
			}
			for _, member := range out.Data.Items {
				if member.OpenID == "" || member.Status == nil || member.Name == "" {
					return nil, &Error{"directory_members", "missing_member_fields"}
				}
				if !seenMembers[member.OpenID] {
					seenMembers[member.OpenID] = true
					members = append(members, member)
				}
			}
			if !out.Data.HasMore {
				break
			}
			if out.Data.Next == "" || seenPages[out.Data.Next] {
				return nil, &Error{"directory_members", "invalid_pagination"}
			}
			seenPages[out.Data.Next] = true
			q.Set("page_token", out.Data.Next)
		}
	}
	return members, nil
}
