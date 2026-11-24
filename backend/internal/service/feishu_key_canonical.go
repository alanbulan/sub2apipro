package service

import "context"

// ConfiguredGroupIDs retains the Pro response contract on the canonical
// single-group model. Upgrade migration checks and backfills legacy edges.
func (k *APIKey) ConfiguredGroupIDs() []int64 {
	if k == nil || k.GroupID == nil || *k.GroupID <= 0 {
		return nil
	}
	return []int64{*k.GroupID}
}

func (s *FeishuKeyDeliveryService) invalidateBalance(ctx context.Context, id int64) error {
	if s.billingCache == nil {
		return nil
	}
	return s.billingCache.InvalidateUserBalance(ctx, id)
}
