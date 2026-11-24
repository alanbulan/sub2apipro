//go:build unit

package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/stretchr/testify/require"
)

func TestFeishuKeyAPINeedsNeitherSMTPNorEmailAndNeverQueuesMail(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	f.service.directory.setup.settings.settingRepo.(*feishuSettingsRepo).values[SettingKeySMTPHost] = ""
	f.members["ou_employee"].EnterpriseEmail, f.members["ou_employee"].Email = "", ""
	first, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
	require.NoError(t, err)
	require.Equal(t, 1, first.Created)
	require.NotEmpty(t, first.Items[0].Key)
	require.Equal(t, 10.0, first.Items[0].InitialGrant)
	require.Equal(t, 10.0, first.Items[0].Balance)
	require.Equal(t, []int64{f.groupID}, first.Items[0].GroupIDs)
	second, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserIDs: []int64{f.userID, f.userID}})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, 1, second.Reused)
	require.Zero(t, second.Items[0].InitialGrant)
	require.Equal(t, first.Items[0].Key, second.Items[0].Key)
	require.Equal(t, 1, f.client.APIKey.Query().CountX(ctx))
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
	require.Empty(t, f.mails)
	require.Zero(t, f.client.Setting.Query().Where(setting.KeyHasPrefix(feishuKeyJobPrefix)).CountX(ctx))
	for _, entry := range f.client.Setting.Query().AllX(ctx) {
		require.NotContains(t, entry.Value, first.Items[0].Key)
	}
}

func TestFeishuKeyAPIAndEmailShareKeyAndOnlyOneInitialGift(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	api, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
	require.NoError(t, err)
	require.Equal(t, 1, api.Created)
	preview, err := f.service.Preview(ctx, 1, nil, true)
	require.NoError(t, err)
	require.True(t, preview.Rows[0].Reuse)
	require.Zero(t, preview.TotalGrant)
	_, err = f.service.Start(ctx, 1, preview.ID)
	require.NoError(t, err)
	f.service.workOne(ctx)
	f.service.workOne(ctx)
	require.Len(t, f.mails, 1)
	require.Contains(t, f.mails[0], api.Items[0].Key)
	require.Equal(t, 10.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
	require.Equal(t, 1, f.client.APIKey.Query().CountX(ctx))
	api, err = f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
	require.NoError(t, err)
	require.Equal(t, 1, api.Reused)
	require.Len(t, f.mails, 1)
}

func TestFeishuKeyAPIBatchReportsUnknownUsersAndRollsBackFailedCreation(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	f.repo.failCreate = true
	out, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserIDs: []int64{f.userID, 99999}})
	require.NoError(t, err)
	require.Equal(t, 2, out.Failed)
	require.Equal(t, "service_error", out.Items[0].Reason)
	require.Equal(t, "identity_mismatch", out.Items[1].Reason)
	require.Empty(t, out.Items[0].Key)
	require.Empty(t, out.Items[1].Key)
	require.Zero(t, f.client.APIKey.Query().CountX(ctx))
	require.Zero(t, f.client.RedeemCode.Query().CountX(ctx))
	require.Zero(t, f.client.User.GetX(ctx, f.userID).Balance)
	require.Zero(t, f.client.User.GetX(ctx, f.userID).QueryAllowedGroups().CountX(ctx))
	f.repo.failCreate = false
	out, err = f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserIDs: []int64{f.userID, 99999}})
	require.NoError(t, err)
	require.Equal(t, 1, out.Created)
	require.Equal(t, 1, out.Failed)
	require.NotEmpty(t, out.Items[0].Key)
	require.Empty(t, out.Items[1].Key)
}

func TestFeishuKeyAPIConcurrentCallsDoNotDuplicateKeyOrBalance(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan *FeishuKeyIssueResult, 4)
	errors := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
			results <- result
			errors <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	created, reused, key := 0, 0, ""
	for result := range results {
		require.Zero(t, result.Failed)
		created, reused = created+result.Created, reused+result.Reused
		if key == "" {
			key = result.Items[0].Key
		}
		require.Equal(t, key, result.Items[0].Key)
	}
	require.Equal(t, 1, created)
	require.Equal(t, 3, reused)
	require.Equal(t, 10.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
	require.Empty(t, f.mails)
}

func TestFeishuKeyAPIGroupOverrideWithoutMailSettingsAndRejectsRevokedKey(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	f.client.Setting.Delete().Where(setting.KeyEQ(feishuKeyConfigKey)).ExecX(ctx)
	f.client.User.UpdateOneID(f.userID).SetBalance(5).ExecX(ctx)
	issued, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID, GroupID: f.groupID})
	require.NoError(t, err)
	require.Equal(t, 1, issued.Created)
	require.Zero(t, issued.Items[0].InitialGrant)
	require.Equal(t, 5.0, issued.Items[0].Balance)
	f.client.APIKey.UpdateOneID(issued.Items[0].KeyID).SetStatus(StatusDisabled).ExecX(ctx)
	issued, err = f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID, GroupID: f.groupID})
	require.NoError(t, err)
	require.Equal(t, 1, issued.Failed)
	require.Equal(t, "key_unavailable", issued.Items[0].Reason)
	require.Empty(t, issued.Items[0].Key)
	require.Equal(t, 1, f.client.APIKey.Query().CountX(ctx))
	require.Zero(t, f.client.RedeemCode.Query().CountX(ctx))
}

func TestFeishuKeyAPIEmployeesListsUnsyncedAndUnissuedWithoutCreatingUsers(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	newEmployee := *f.members["ou_employee"]
	newEmployee.OpenID, newEmployee.Name, newEmployee.EnterpriseEmail = "ou_new", "New employee", "new@company.example"
	f.members[newEmployee.OpenID] = &newEmployee
	unissued := false
	listed, err := f.service.ListEmployees(ctx, 1, 1, &unissued, "")
	require.NoError(t, err)
	require.Equal(t, 2, listed.Total)
	require.Equal(t, 2, listed.Pages)
	require.Zero(t, listed.Items[0].UserID)
	require.False(t, listed.Items[0].Synced)
	listed, err = f.service.ListEmployees(ctx, 2, 1, &unissued, "")
	require.NoError(t, err)
	require.Equal(t, f.userID, listed.Items[0].UserID)
	require.True(t, listed.Items[0].Synced)
	require.False(t, listed.Items[0].KeyIssued)
	issued, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
	require.NoError(t, err)
	listed, err = f.service.ListEmployees(ctx, 1, 50, &unissued, "")
	require.NoError(t, err)
	require.Equal(t, 1, listed.Total)
	require.False(t, listed.Items[0].Synced)
	flag := true
	listed, err = f.service.ListEmployees(ctx, 1, 50, &flag, "employee@company.example")
	require.NoError(t, err)
	require.Equal(t, 1, listed.Total)
	require.Equal(t, issued.Items[0].KeyID, listed.Items[0].KeyID)
	raw, err := json.Marshal(listed)
	require.NoError(t, err)
	require.NotContains(t, string(raw), issued.Items[0].Key)
	require.Equal(t, 1, f.client.User.Query().CountX(ctx))
	require.Empty(t, f.mails)
}

func TestFeishuKeyAPIRejectsInvalidScopeAndInactiveEmployee(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	for _, req := range []FeishuKeyIssueRequest{{}, {UserID: -1}, {UserIDs: []int64{0}}, {UserID: f.userID, UserIDs: []int64{f.userID}}, {UserID: f.userID, GroupID: -1}, {UserIDs: make([]int64, 101)}} {
		_, err := f.service.IssueKeys(ctx, 1, req)
		require.Error(t, err)
	}
	f.members["ou_employee"].Status.Resigned = true
	out, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
	require.NoError(t, err)
	require.Equal(t, "inactive_employee", out.Items[0].Reason)
	require.Empty(t, out.Items[0].Key)
	require.Zero(t, f.client.APIKey.Query().CountX(ctx))
}

func TestFeishuKeyAPIExplicitGroupCannotSilentlyReassignExistingKey(t *testing.T) {
	f := newFeishuKeyFixture(t)
	ctx := context.Background()
	first, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID})
	require.NoError(t, err)
	require.Equal(t, 1, first.Created)
	other := f.client.Group.Create().SetName("Other pool").SetPlatform(PlatformOpenAI).SetStatus(StatusActive).SaveX(ctx)
	second, err := f.service.IssueKeys(ctx, 1, FeishuKeyIssueRequest{UserID: f.userID, GroupID: other.ID})
	require.NoError(t, err)
	require.Equal(t, 1, second.Failed)
	require.Equal(t, "group_mismatch", second.Items[0].Reason)
	require.Empty(t, second.Items[0].Key)
	require.Equal(t, 1, f.client.APIKey.Query().CountX(ctx))
	require.Equal(t, 10.0, f.client.User.GetX(ctx, f.userID).Balance)
	require.Equal(t, 1, f.client.RedeemCode.Query().CountX(ctx))
}
