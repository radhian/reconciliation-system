package reconciliation

import (
	"context"
	"log"

	"github.com/radhian/reconciliation-system/infra/db/model"
)

func (u *reconciliationUsecase) TryAcquireLock(ctx context.Context) (bool, int64, error) {
	var processLogList []model.ReconciliationProcessLog

	processLogList, err := u.dao.GetReconciliationProcessLogByStatusList([]int{1, 2})
	if err != nil {
		return false, 0, err
	}

	for _, processLog := range processLogList {
		if u.locker.IsProcessing(processLog.ID) {
			continue
		}

		u.locker.MarkAsProcessing(processLog.ID)
		log.Printf("[LOCK_PROCESS] log_id:%d", processLog.ID)
		return true, processLog.ID, nil
	}

	return false, 0, nil
}

func (u *reconciliationUsecase) UnlockProcess(ctx context.Context, logsID int64) {
	u.locker.Unlock(logsID)
	log.Printf("[UNLOCK_PROCESS] log_id:%d", logsID)
}
