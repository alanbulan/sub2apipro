package service

import "context"

func (s *ScheduledTestService) GetResult(ctx context.Context, planID, resultID int64) (*ScheduledTestResult, error) {
	return s.resultRepo.GetResult(ctx, planID, resultID)
}

// ListPelicanHistory pages retained Pelican outputs; accountID 0 lists every account.
func (s *ScheduledTestService) ListPelicanHistory(ctx context.Context, beforeID int64, limit int, accountID int64) (*PelicanHistoryPage, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	items, err := s.resultRepo.ListPelicanHistory(ctx, beforeID, limit+1, accountID)
	if err != nil {
		return nil, err
	}
	page := &PelicanHistoryPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		page.NextCursor = items[limit-1].ID
	}
	return page, nil
}
