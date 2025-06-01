package reconciliation

import (
	"context"

	"github.com/radhian/reconciliation-system/infra/db/dao"
	"github.com/radhian/reconciliation-system/infra/db/model"
	"github.com/radhian/reconciliation-system/infra/locker"
)

type ReconciliationUsecase interface {
	ProcessReconciliationInit(transactionCSV string, referenceCSVs []string, startTime, endTime int64, operator string) (*model.ReconciliationProcessLog, error)
	GetReconciliationResults() ([]model.ReconciliationProcessLog, error)
	ProcessReconciliationJob(ctx context.Context, logID int64) error
	TryAcquireLock(ctx context.Context) (bool, int64, error)
	UnlockProcess(ctx context.Context, logsID int64)
}

type reconciliationUsecase struct {
	dao       dao.DaoMethod
	locker    *locker.Locker
	batchSize int64
}

func NewReconciliationUsecase(dao dao.DaoMethod, locker *locker.Locker, batchSize int64) ReconciliationUsecase {
	return &reconciliationUsecase{dao: dao, locker: locker, batchSize: batchSize}
}
