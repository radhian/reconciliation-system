package reconciliation

import (
	"context"
)

func (u *reconciliationUsecase) TryAcquireLock(ctx context.Context) (bool, error) {
	return true, nil
}
