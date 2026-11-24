//go:build unit

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/feishu"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type feishuKeyTestUsers struct {
	UserRepository
	client *dbent.Client
}

func feishuTestClient(ctx context.Context, c *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return c
}
func (r *feishuKeyTestUsers) GetByID(ctx context.Context, id int64) (*User, error) {
	c := feishuTestClient(ctx, r.client)
	u, err := c.User.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	ids, err := u.QueryAllowedGroups().IDs(ctx)
	if err != nil {
		return nil, err
	}
	return &User{ID: u.ID, Email: u.Email, Username: u.Username, Role: u.Role, Status: u.Status, Balance: u.Balance, SignupSource: u.SignupSource, AllowedGroups: ids, RestrictPublicGroups: u.RestrictPublicGroups}, nil
}
func (r *feishuKeyTestUsers) FillFeishuContactEmail(ctx context.Context, id int64, email string) (bool, error) {
	c := feishuTestClient(ctx, r.client)
	exists, err := c.User.Query().Where(dbuser.EmailEQ(email), dbuser.IDNEQ(id)).Exist(ctx)
	if err != nil {
		return false, err
	}
	if exists {
		return false, ErrEmailExists
	}
	n, err := c.User.Update().Where(dbuser.IDEQ(id), dbuser.SignupSourceEQ("feishu"), dbuser.EmailHasSuffix(FeishuConnectSyntheticEmailDomain)).SetEmail(email).Save(ctx)
	return n > 0, err
}

type feishuKeyTestGroups struct {
	GroupRepository
	client *dbent.Client
}

func (r *feishuKeyTestGroups) GetByID(ctx context.Context, id int64) (*Group, error) {
	g, err := feishuTestClient(ctx, r.client).Group.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Group{ID: g.ID, Name: g.Name, Status: g.Status, IsExclusive: g.IsExclusive, SubscriptionType: g.SubscriptionType}, nil
}

type feishuKeyTestKeys struct {
	APIKeyRepository
	client     *dbent.Client
	failCreate bool
}

func (r *feishuKeyTestKeys) Create(ctx context.Context, k *APIKey) error {
	v, err := feishuTestClient(ctx, r.client).APIKey.Create().SetUserID(k.UserID).SetName(k.Name).SetKey(k.Key).SetStatus(k.Status).SetNillableGroupID(k.GroupID).SetQuota(k.Quota).SetNillableExpiresAt(k.ExpiresAt).Save(ctx)
	if err != nil {
		return err
	}
	k.ID = v.ID
	if r.failCreate {
		return errors.New("injected persistence failure after key insert")
	}
	return nil
}
func (r *feishuKeyTestKeys) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	k, err := r.client.APIKey.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if k.DeletedAt != nil {
		return nil, ErrAPIKeyNotFound
	}
	return &APIKey{ID: k.ID, UserID: k.UserID, Key: k.Key, Name: k.Name, Status: k.Status, GroupID: k.GroupID, Quota: k.Quota, QuotaUsed: k.QuotaUsed, ExpiresAt: k.ExpiresAt, IPWhitelist: k.IPWhitelist, IPBlacklist: k.IPBlacklist}, nil
}
func (r *feishuKeyTestKeys) ListKeysByUserID(ctx context.Context, id int64) ([]string, error) {
	return r.client.APIKey.Query().Where(apikey.UserIDEQ(id)).Select(apikey.FieldKey).Strings(ctx)
}

type feishuKeyFixture struct {
	service         *FeishuKeyDeliveryService
	client          *dbent.Client
	members         map[string]*feishu.DirectoryMember
	repo            *feishuKeyTestKeys
	mails           []string
	userID, groupID int64
}

func newFeishuKeyFixture(t *testing.T) *feishuKeyFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:feishu_key_%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", time.Now().UnixNano()))
	require.NoError(t, err)
	c := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = c.Close() })
	setup, _, settingsRepo, _ := newFeishuSetupTest(t)
	for k, v := range map[string]string{SettingKeyFeishuConnectEnabled: "true", SettingKeyFeishuConnectAppID: "cli_employee", SettingKeyFeishuConnectAppSecret: "test-secret", SettingKeyFeishuConnectTenantKey: "tenant_1", SettingKeyFeishuConnectRedirectURL: "https://example.com" + feishu.CallbackPath, SettingKeySMTPHost: "smtp.example.com", SettingKeySMTPPort: "587", SettingKeySMTPFrom: "sender@example.com", SettingKeySMTPUsername: "sender", SettingKeySMTPPassword: "test-only"} {
		settingsRepo.values[k] = v
	}
	u := c.User.Create().SetEmail("placeholder" + FeishuConnectSyntheticEmailDomain).SetPasswordHash("test-hash").SetUsername("Employee").SetSignupSource("feishu").SetRole(RoleUser).SetStatus(StatusActive).SaveX(ctx)
	g := c.Group.Create().SetName("Employee pool").SetPlatform(PlatformOpenAI).SetStatus(StatusActive).SetIsExclusive(true).SaveX(ctx)
	c.AuthIdentity.Create().SetUserID(u.ID).SetProviderType("feishu").SetProviderKey("feishu:cli_employee:tenant_1").SetProviderSubject("ou_employee").SaveX(ctx)
	ur := &feishuKeyTestUsers{client: c}
	kr := &feishuKeyTestKeys{client: c}
	auth := &AuthService{entClient: c, userRepo: ur, settingService: setup.settings}
	keys := &APIKeyService{apiKeyRepo: kr, userRepo: ur, groupRepo: &feishuKeyTestGroups{client: c}, cfg: &config.Config{}}
	s := NewFeishuKeyDeliveryService(NewFeishuDirectoryService(auth, setup), keys, NewEmailService(settingsRepo, nil))
	f := &feishuKeyFixture{service: s, client: c, repo: kr, members: map[string]*feishu.DirectoryMember{}, userID: u.ID, groupID: g.ID}
	var m feishu.DirectoryMember
	require.NoError(t, json.Unmarshal([]byte(`{"open_id":"ou_employee","name":"Employee","enterprise_email":"employee@company.example","email":"contact@company.example","status":{"is_activated":true}}`), &m))
	f.members[m.OpenID] = &m
	setup.api = &feishu.Client{HTTP: &http.Client{Transport: feishuTransport(func(r *http.Request) (*http.Response, error) {
		var out any = map[string]any{"code": 0}
		switch {
		case strings.HasSuffix(r.URL.Path, "/tenant_access_token/internal"):
			out = map[string]any{"code": 0, "tenant_access_token": "tenant-token"}
		case strings.HasSuffix(r.URL.Path, "/departments/0/children"):
			out = map[string]any{"code": 0, "data": map[string]any{"items": []any{}, "has_more": false}}
		case strings.HasSuffix(r.URL.Path, "/find_by_department"):
			members := []*feishu.DirectoryMember{}
			for _, m := range f.members {
				members = append(members, m)
			}
			out = map[string]any{"code": 0, "data": map[string]any{"items": members, "has_more": false}}
		default:
			out = map[string]any{"code": 0, "data": map[string]any{"user": f.members[strings.TrimPrefix(r.URL.Path, "/open-apis/contact/v3/users/")]}}
		}
		body, _ := json.Marshal(out)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}}
	s.send = func(_ context.Context, to, subject, body string) error {
		f.mails = append(f.mails, to+"\n"+subject+"\n"+body)
		return nil
	}
	_, err = s.SaveConfig(ctx, FeishuKeyDeliveryConfig{GroupIDs: []int64{g.ID}, InitialBalance: 10, Quota: 100, NameSuffix: "Enterprise", Subject: "Your key", APIURL: "https://api.example.com/v1"})
	require.NoError(t, err)
	return f
}

func TestFeishuKeyDeliveryFirstIssueAndCrossBatchReuse(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	s := f.service
	p, err := s.Preview(ctx, 1, nil, true)
	require.NoError(t, err)
	require.Equal(t, 1, p.Ready)
	require.Equal(t, 10.0, p.TotalGrant)
	require.Equal(t, "employee@company.example", p.Rows[0].Email)
	require.Zero(t, f.client.APIKey.Query().CountX(ctx))
	require.Equal(t, 0.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Empty(t, f.mails)
	// Concurrent double clicks identify the same persisted preview, not two jobs.
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Start(ctx, 1, p.ID); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	s.workOne(ctx)
	s.workOne(ctx)
	current, err := s.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, "completed", current.Status)
	require.Equal(t, 1, current.Sent)
	key := f.client.APIKey.Query().OnlyX(ctx)
	require.Len(t, f.mails, 1)
	require.Contains(t, f.mails[0], key.Key)
	require.Equal(t, 10.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
	for _, entry := range f.client.Setting.Query().AllX(ctx) {
		require.NotContains(t, entry.Value, key.Key, "keys must not be copied into progress/configuration storage")
	}
	p, err = s.Preview(ctx, 1, []int64{f.userID}, false)
	require.NoError(t, err)
	require.True(t, p.Rows[0].Reuse)
	require.Zero(t, p.TotalGrant)
	_, err = s.Start(ctx, 1, p.ID)
	require.NoError(t, err)
	s.workOne(ctx)
	s.workOne(ctx)
	require.Equal(t, 1, f.client.APIKey.Query().CountX(ctx))
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
	require.Equal(t, 10.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Len(t, f.mails, 2)
	require.Contains(t, f.mails[1], key.Key)
}

func TestFeishuKeyDeliveryRollbackAndFailedSMTPReuse(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	s := f.service
	p, err := s.Preview(ctx, 1, nil, true)
	require.NoError(t, err)
	_, err = s.Start(ctx, 1, p.ID)
	require.NoError(t, err)
	f.repo.failCreate = true
	s.workOne(ctx)
	s.workOne(ctx)
	require.Zero(t, f.client.APIKey.Query().CountX(ctx))
	require.Zero(t, f.client.RedeemCode.Query().CountX(ctx))
	require.Equal(t, 0.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Zero(t, f.client.User.GetX(ctx, f.userID).QueryAllowedGroups().CountX(ctx))
	f.repo.failCreate = false
	s.send = func(context.Context, string, string, string) error { return errors.New("smtp auth: rejected") }
	_, err = s.Retry(ctx, 1, p.ID, false)
	require.NoError(t, err)
	s.workOne(ctx)
	s.workOne(ctx)
	key := f.client.APIKey.Query().OnlyX(ctx)
	require.Equal(t, 10.0, f.client.User.GetX(ctx, f.userID).Balance)
	s.send = func(_ context.Context, _, _, body string) error { require.Contains(t, body, key.Key); return nil }
	_, err = s.Retry(ctx, 1, p.ID, false)
	require.NoError(t, err)
	s.workOne(ctx)
	s.workOne(ctx)
	current, err := s.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, current.Sent)
	require.Equal(t, 1, f.client.APIKey.Query().CountX(ctx))
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
}

func TestFeishuKeyDeliveryRestartNeverBlindlyResendsSMTP(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	s := f.service
	p, err := s.Preview(ctx, 1, nil, true)
	require.NoError(t, err)
	_, err = s.Start(ctx, 1, p.ID)
	require.NoError(t, err)
	// Simulate a process dying after SMTP DATA began, before recording acceptance.
	var b feishuKeyBatch
	_, err = feishuReadJSON(ctx, f.client, feishuKeyJobPrefix+p.ID, &b)
	require.NoError(t, err)
	b.Rows[0].Status = "sending"
	b.Worker = "old-process"
	b.LeaseUntil = time.Now().Add(-time.Second).Unix()
	require.NoError(t, feishuWriteJSON(ctx, f.client, feishuKeyJobPrefix+b.ID, &b))
	s.workOne(ctx)
	current, err := s.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, current.Uncertain)
	require.Empty(t, f.mails)
	_, err = s.Retry(ctx, 1, p.ID, false)
	require.Error(t, err)
	_, err = s.Retry(ctx, 1, p.ID, true)
	require.NoError(t, err)
}

func TestFeishuKeyDeliveryRechecksMailboxAndMembership(t *testing.T) {
	for _, scenario := range []string{"email_changed", "resigned", "unbound", "revoked"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFeishuKeyFixture(t)
			ctx := context.Background()
			s := f.service
			p, err := s.Preview(ctx, 1, nil, true)
			require.NoError(t, err)
			_, err = s.Start(ctx, 1, p.ID)
			require.NoError(t, err)
			switch scenario {
			case "email_changed":
				f.members["ou_employee"].EnterpriseEmail = "changed@company.example"
			case "resigned":
				f.members["ou_employee"].Status.Resigned = true
			case "unbound":
				f.client.AuthIdentity.Delete().ExecX(ctx)
			case "revoked":
				f.client.User.UpdateOneID(f.userID).SetStatus("disabled").ExecX(ctx)
			}
			s.workOne(ctx)
			s.workOne(ctx)
			current, err := s.Current(ctx)
			require.NoError(t, err)
			require.Equal(t, 1, current.Failed)
			require.Empty(t, f.mails)
			require.Zero(t, f.client.APIKey.Query().CountX(ctx))
			require.Zero(t, f.client.RedeemCode.Query().CountX(ctx))
		})
	}
}

func TestFeishuEmailBackfillUsesExistingIdentityAndProtectsContacts(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	s := f.service
	s.backfillEmails(ctx)
	require.Equal(t, "employee@company.example", f.client.User.GetX(ctx, f.userID).Email)
	require.Zero(t, f.client.APIKey.Query().CountX(ctx))
	require.Empty(t, f.mails)
	// Once filled, imported contact data never creates a verified email identity.
	require.Equal(t, 1, f.client.AuthIdentity.Query().CountX(ctx))
	f.client.User.UpdateOneID(f.userID).SetEmail("bound@company.example").ExecX(ctx)
	s.backfillEmails(ctx)
	require.Equal(t, "bound@company.example", f.client.User.GetX(ctx, f.userID).Email)
}

func TestFeishuKeyDeliveryTemplateEscapesContent(t *testing.T) {
	body, err := renderFeishuKeyEmail(FeishuKeyDeliveryConfig{Introduction: "<img src=x onerror=alert(1)>", APIURL: "https://api.example.com/v1"}, FeishuKeyDeliveryRow{Name: "<script>bad</script>", GroupNames: []string{"<b>Group</b>"}}, "sk-only-recipient")
	require.NoError(t, err)
	require.NotContains(t, body, "<script>")
	require.NotContains(t, body, "<img src=x")
	require.Contains(t, body, "&lt;script&gt;")
	require.Contains(t, body, "sk-only-recipient")
}

func TestFeishuKeyDeliveryEnterpriseMailboxIsDefault(t *testing.T) {
	m := feishu.DirectoryMember{Email: "contact@company.example"}
	require.Empty(t, feishuRecipientEmail(m, false))
	require.Equal(t, m.Email, feishuRecipientEmail(m, true))
	m.EnterpriseEmail = "STAFF@company.example"
	require.Equal(t, "staff@company.example", feishuRecipientEmail(m, true))
}
