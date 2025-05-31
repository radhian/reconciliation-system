package reconciliation

import (
	"github.com/radhian/reconciliation-system/infra/db/model"

	"github.com/jinzhu/gorm"
)

type ReconciliationUsecase interface {
	ProcessReconciliationInit(transactionCSV string, referenceCSVs []string, startTime, endTime int64, operator string) (*model.ReconciliationProcessLog, error)
	GetReconciliationResults() ([]model.ReconciliationProcessLog, error)
}

type reconciliationUsecase struct {
	db *gorm.DB
}

func NewReconciliationUsecase(db *gorm.DB) ReconciliationUsecase {
	return &reconciliationUsecase{db: db}
}
