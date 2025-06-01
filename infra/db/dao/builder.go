package dao

import (
	"github.com/radhian/reconciliation-system/infra/db/model"

	"github.com/jinzhu/gorm"
)

type DaoMethod interface {
	GetReconciliationProcessLog() ([]model.ReconciliationProcessLog, error)
	GetReconciliationProcessLogByStatusList(statusList []int) ([]model.ReconciliationProcessLog, error)
	CreateReconciliationProcessLog(payloadList model.ReconciliationProcessLog) error
	CreateReconciliationProcessLogAsset(payload model.ReconciliationProcessLogAsset) error
	GetReconciliationProcessLogByID(logID uint) (model.ReconciliationProcessLog, error)
	GetReconciliationLogAssetsByLogID(logID uint) ([]model.ReconciliationProcessLogAsset, error)
	UpdateReconciliationProcessLog(logEntry model.ReconciliationProcessLog) error
}

type dao struct {
	db *gorm.DB
}

func NewDaoMethod(db *gorm.DB) DaoMethod {
	return &dao{db: db}
}
