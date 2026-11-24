package service

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
)

type FeishuKeyIssueRequest struct {
	UserID  int64   `json:"user_id"`
	UserIDs []int64 `json:"user_ids"`
	GroupID int64   `json:"group_id"`
}

type FeishuKeyIssueItem struct {
	UserID       int64      `json:"user_id"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	Key          string     `json:"key,omitempty"`
	KeyID        int64      `json:"key_id,omitempty"`
	GroupIDs     []int64    `json:"group_ids,omitempty"`
	Balance      float64    `json:"balance"`
	InitialGrant float64    `json:"initial_grant"`
	Quota        float64    `json:"quota"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type FeishuKeyIssueResult struct {
	Items   []FeishuKeyIssueItem `json:"items"`
	Created int                  `json:"created"`
	Reused  int                  `json:"reused"`
	Failed  int                  `json:"failed"`
}

type FeishuKeyEmployee struct {
	UserID          int64      `json:"user_id"`
	Name            string     `json:"name"`
	EnterpriseEmail string     `json:"enterprise_email"`
	Synced          bool       `json:"synced"`
	KeyIssued       bool       `json:"key_issued"`
	KeyID           int64      `json:"key_id,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}

type FeishuKeyEmployeeList struct {
	Items    []FeishuKeyEmployee `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Pages    int                 `json:"pages"`
}

func (s *FeishuKeyDeliveryService) employeeSnapshot(ctx context.Context) (*FeishuConnectSettings, []feishu.DirectoryMember, []*dbent.AuthIdentity, error) {
	if !s.available() {
		return nil, nil, nil, ErrServiceUnavailable
	}
	cfg, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	token, err := s.directory.setup.api.TenantToken(ctx, cfg.AppID, cfg.AppSecret)
	if err != nil {
		return nil, nil, nil, err
	}
	members, err := s.directory.setup.api.DirectoryMembers(ctx, token)
	if err != nil {
		return nil, nil, nil, err
	}
	identities, err := s.client.AuthIdentity.Query().Where(authidentity.ProviderTypeEQ("feishu"), authidentity.ProviderKeyEQ(cfg.IdentityKey())).All(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	current, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
	if err != nil || current.Fingerprint() != cfg.Fingerprint() {
		return nil, nil, nil, ErrFeishuKeyStale
	}
	return cfg, members, identities, nil
}

// ListEmployees includes live active employees who have not signed in/synced
// yet (user_id=0), without creating users or issuing keys as a GET side effect.
// key_issued describes our durable enterprise issuance receipt, not hire date.
func (s *FeishuKeyDeliveryService) ListEmployees(ctx context.Context, page, pageSize int, issued *bool, search string) (*FeishuKeyEmployeeList, error) {
	if page < 1 || page > 1000000 || pageSize < 1 || pageSize > 100 || len(search) > 200 {
		return nil, ErrFeishuKeyInvalid
	}
	_, members, identities, err := s.employeeSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	owners := map[string]int64{}
	ids := []int64{}
	for _, identity := range identities {
		owners[identity.ProviderSubject] = identity.UserID
		ids = append(ids, identity.UserID)
	}
	users := map[int64]*dbent.User{}
	receipts := map[int64]feishuKeyReceipt{}
	if len(ids) > 0 {
		rows, err := s.client.User.Query().Where(dbuser.IDIn(ids...), dbuser.DeletedAtIsNil(), dbuser.StatusEQ(StatusActive), dbuser.RoleEQ(RoleUser)).All(ctx)
		if err != nil {
			return nil, err
		}
		keys := []string{}
		for _, u := range rows {
			users[u.ID] = u
			keys = append(keys, feishuKeyReceiptKey(u.ID))
		}
		if len(keys) > 0 {
			stored, err := s.client.Setting.Query().Where(setting.KeyIn(keys...)).All(ctx)
			if err != nil {
				return nil, err
			}
			for _, entry := range stored {
				var receipt feishuKeyReceipt
				if json.Unmarshal([]byte(entry.Value), &receipt) != nil {
					return nil, ErrServiceUnavailable
				}
				receipts[receipt.UserID] = receipt
			}
		}
	}
	items := []FeishuKeyEmployee{}
	search = strings.ToLower(strings.TrimSpace(search))
	for _, m := range members {
		if !m.Active() {
			continue
		}
		id := owners[m.OpenID]
		u := users[id]
		if id != 0 && u == nil { // Disabled/deleted users and administrators.
			continue
		}
		r := FeishuKeyEmployee{UserID: id, Name: m.Name, EnterpriseEmail: validFeishuContactEmail(m.EnterpriseEmail), Synced: u != nil}
		if u != nil {
			r.CreatedAt = &u.CreatedAt
		}
		if receipt, ok := receipts[id]; ok {
			r.KeyIssued, r.KeyID = true, receipt.KeyID
		}
		if issued != nil && r.KeyIssued != *issued {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(r.Name+" "+r.EnterpriseEmail+" "+strconv.FormatInt(id, 10)), search) {
			continue
		}
		items = append(items, r)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UserID != items[j].UserID {
			return items[i].UserID < items[j].UserID
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].EnterpriseEmail < items[j].EnterpriseEmail
	})
	total := len(items)
	start := min((page-1)*pageSize, total)
	end := min(start+pageSize, total)
	return &FeishuKeyEmployeeList{Items: items[start:end], Total: total, Page: page, PageSize: pageSize, Pages: (total + pageSize - 1) / pageSize}, nil
}

// IssueKeys returns secrets only to the authenticated administrator. It never
// creates an email job or calls SMTP. Each employee is an independent atomic
// transaction; the receipt is shared with the existing email worker.
func (s *FeishuKeyDeliveryService) IssueKeys(ctx context.Context, actor int64, request FeishuKeyIssueRequest) (*FeishuKeyIssueResult, error) {
	if !s.available() {
		return nil, ErrServiceUnavailable
	}
	if actor <= 0 || request.GroupID < 0 || request.UserID != 0 && len(request.UserIDs) > 0 {
		return nil, ErrFeishuKeyInvalid
	}
	ids := request.UserIDs
	if request.UserID != 0 {
		ids = []int64{request.UserID}
	}
	if len(ids) == 0 || len(ids) > 100 {
		return nil, ErrFeishuKeyInvalid
	}
	selected, seen := []int64{}, map[int64]bool{}
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrFeishuKeyInvalid
		}
		if !seen[id] {
			selected, seen[id] = append(selected, id), true
		}
	}
	settings, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if request.GroupID > 0 {
		settings.Config.GroupIDs = []int64{request.GroupID}
	}
	if err = validateFeishuKeyProvisionConfig(&settings.Config); err != nil {
		return nil, err
	}
	cfg, members, identities, err := s.employeeSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	byID, ownerCount := map[int64]string{}, map[int64]int{}
	for _, identity := range identities {
		byID[identity.UserID] = identity.ProviderSubject
		ownerCount[identity.UserID]++
	}
	byOpenID := map[string]feishu.DirectoryMember{}
	for _, m := range members {
		byOpenID[m.OpenID] = m
	}
	batchID, err := randomHexString(20)
	if err != nil {
		return nil, err
	}
	b := &feishuKeyBatch{ID: "api-" + batchID, Config: settings.Config, Fingerprint: cfg.Fingerprint(), ProviderKey: cfg.IdentityKey(), ActorID: actor}
	out := &FeishuKeyIssueResult{Items: []FeishuKeyIssueItem{}}
	for _, id := range selected {
		item := FeishuKeyIssueItem{UserID: id, Status: "failed"}
		m, ok := byOpenID[byID[id]]
		if !ok || ownerCount[id] != 1 {
			item.Reason = "identity_mismatch"
		} else {
			item.Name = m.Name
			r := &feishuKeyRecipient{OpenID: m.OpenID, FeishuKeyDeliveryRow: FeishuKeyDeliveryRow{UserID: id, Name: m.Name}}
			err = s.transaction(ctx, func(ctx context.Context, c *dbent.Client) error {
				current, err := s.directory.setup.settings.GetFeishuOAuthConfig(ctx)
				if err != nil || current.Fingerprint() != b.Fingerprint {
					item.Reason = "config_changed"
					return nil
				}
				verified, err := c.AuthIdentity.Query().Where(authidentity.UserIDEQ(id), authidentity.ProviderTypeEQ("feishu"), authidentity.ProviderKeyEQ(b.ProviderKey), authidentity.ProviderSubjectEQ(m.OpenID)).Exist(ctx)
				if err != nil {
					return err
				}
				if !verified {
					item.Reason = "identity_mismatch"
					return nil
				}
				if item.Reason = s.evaluateEmployee(ctx, c, b, r, m, true); item.Reason != "" {
					return nil
				}
				if r.Reuse && request.GroupID > 0 {
					key, err := s.keys.apiKeyRepo.GetByID(ctx, r.KeyID)
					if err != nil {
						return err
					}
					groups := key.ConfiguredGroupIDs()
					if len(groups) != 1 || groups[0] != request.GroupID {
						item.Reason = "group_mismatch"
						return nil
					}
				}
				return s.provisionKey(ctx, c, b, r)
			})
			if err != nil {
				item.Reason = "service_error"
			} else if item.Reason == "" {
				s.keys.InvalidateAuthCacheByUserID(ctx, id)
				if s.keys.autoCreditBilling != nil {
					err = s.keys.autoCreditBilling.InvalidateUserBalance(ctx, id)
				}
				key, keyErr := s.keys.apiKeyRepo.GetByID(ctx, r.KeyID)
				if err != nil {
					item.Reason = "service_error"
				} else if keyErr != nil || key == nil || key.UserID != id || !key.IsActive() || key.IsExpired() || key.IsQuotaExhausted() {
					item.Reason = "key_unavailable"
				} else {
					item.Status = "created"
					if r.Reuse {
						item.Status = "reused"
					}
					item.Key, item.KeyID, item.GroupIDs = key.Key, key.ID, key.ConfiguredGroupIDs()
					item.Balance, item.InitialGrant, item.Quota, item.ExpiresAt = r.Balance, r.Grant, key.Quota, key.ExpiresAt
				}
			}
		}
		switch item.Status {
		case "created":
			out.Created++
		case "reused":
			out.Reused++
		default:
			out.Failed++
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}
