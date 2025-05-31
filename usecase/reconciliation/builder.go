package reconciliation

import (
	"context"

	"github.com/radhian/reconciliation-system/infra/db/model"
	"github.com/radhian/reconciliation-system/infra/locker"

	"github.com/jinzhu/gorm"
)

type ReconciliationUsecase interface {
	ProcessReconciliationInit(transactionCSV string, referenceCSVs []string, startTime, endTime int64, operator string) (*model.ReconciliationProcessLog, error)
	GetReconciliationResults() ([]model.ReconciliationProcessLog, error)
	ProcessReconciliationJob(ctx context.Context, logID int64) error
	TryAcquireLock(ctx context.Context) (bool, int64, error)
	UnlockProcess(ctx context.Context, logsID int64)
}

type reconciliationUsecase struct {
	db     *gorm.DB
	locker *locker.Locker
}

func NewReconciliationUsecase(db *gorm.DB, locker *locker.Locker) ReconciliationUsecase {
	return &reconciliationUsecase{db: db, locker: locker}
}
