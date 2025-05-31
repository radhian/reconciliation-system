package reconciliation

import (
	"github.com/radhian/reconciliation_system/infra/db/model"

	"github.com/jinzhu/gorm"
)

type ReconciliationUsecase interface {
	ProcessReconciliation(transactionCSV string, referenceCSVs []string, startTime, endTime int64, operator string) (*model.ReconciliationProcessLog, error)
	GetReconciliationResults() ([]model.ReconciliationProcessLog, error)
}

type reconciliationUsecase struct {
	db *gorm.DB
}

func NewReconciliationUsecase(db *gorm.DB) ReconciliationUsecase {
	return &reconciliationUsecase{db: db}
}

func (u *reconciliationUsecase) GetReconciliationResults() ([]model.ReconciliationProcessLog, error) {
	var logs []model.ReconciliationProcessLog
	if err := u.db.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
