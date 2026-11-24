package admin

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type poolGroupReader interface {
	PoolSourceGroups(context.Context, int64) ([]service.Group, error)
	GetUserGroupDeniedModels(context.Context, int64) (map[int64][]string, error)
}
type poolQuotaCacheEntry struct {
	expires  time.Time
	accounts []service.PoolQuotaAccount
}
type poolDashboardCache struct {
	mu     sync.Mutex
	groups map[int64]poolQuotaCacheEntry
	tests  poolDiagnosticLimits
}

func (h *AccountHandler) SetPoolGroupReader(reader poolGroupReader) { h.poolGroupReader = reader }

// SharedPoolDashboard runs behind ordinary-user JWT authentication. Authorize
// source groups on EVERY request, before consulting the shared quota cache.
func (h *AccountHandler) SharedPoolDashboard(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.poolGroupReader == nil {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	groups, err := h.poolGroupReader.PoolSourceGroups(ctx, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := service.UserPoolDashboard{GeneratedAt: time.Now().UTC(), Groups: make([]service.PoolQuotaGroup, 0, len(groups))}
	h.poolDashboard.mu.Lock()
	defer h.poolDashboard.mu.Unlock()
	if h.poolDashboard.groups == nil {
		h.poolDashboard.groups = map[int64]poolQuotaCacheEntry{}
	}
	for id, entry := range h.poolDashboard.groups {
		if time.Now().After(entry.expires) {
			delete(h.poolDashboard.groups, id)
		}
	}
	for _, group := range groups {
		if group.ID <= 0 {
			continue
		}
		entry, exists := h.poolDashboard.groups[group.ID]
		if !exists {
			accounts, err := h.adminService.ListAccountsForSchedulerScoreFilter(ctx, "", "", "", "", group.ID, "")
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
			parentIDs := make([]int64, 0)
			seenParents := map[int64]bool{}
			for i := range accounts {
				if parent := accounts[i].ParentAccountID; parent != nil && !seenParents[*parent] {
					seenParents[*parent] = true
					parentIDs = append(parentIDs, *parent)
				}
			}
			parents := map[int64]*service.Account{}
			if len(parentIDs) > 0 {
				rows, err := h.adminService.GetAccountsByIDs(ctx, parentIDs)
				if err != nil {
					response.ErrorFrom(c, err)
					return
				}
				for _, row := range rows {
					if row != nil {
						parents[row.ID] = row
					}
				}
			}
			entry = poolQuotaCacheEntry{expires: time.Now().Add(30 * time.Second), accounts: h.accountUsageService.PoolQuotaAccounts(accounts, parents, time.Now().UTC())}
			h.poolDashboard.groups[group.ID] = entry
		}
		// Group DTOs already expose these public labels to the same authorized user.
		out.Groups = append(out.Groups, service.PoolQuotaGroup{ID: group.ID, Name: group.Name, Platform: group.Platform, Accounts: entry.accounts})
	}
	response.Success(c, out)
}
