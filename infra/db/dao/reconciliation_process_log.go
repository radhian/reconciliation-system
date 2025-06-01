package dao

import (
	"fmt"

	"github.com/radhian/reconciliation-system/infra/db/model"
)

func (d *dao) GetReconciliationProcessLog() ([]model.ReconciliationProcessLog, error) {
	var logs []model.ReconciliationProcessLog
	if err := d.db.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (d *dao) GetReconciliationProcessLogByStatusList(statusList []int) ([]model.ReconciliationProcessLog, error) {
	var processLogList []model.ReconciliationProcessLog
	if err := d.db.
		Select("id").
		Where("status IN (?)", statusList).
		Order("create_time ASC").
		Find(&processLogList).Error; err != nil {
		return nil, err
	}
	return processLogList, nil
}

func (d *dao) CreateReconciliationProcessLog(payload *model.ReconciliationProcessLog) error {
	if err := d.db.Create(payload).Error; err != nil {
		return fmt.Errorf("failed to save file asset: %v", err)
	}
	return nil
}

func (d *dao) GetReconciliationProcessLogByID(logID uint) (model.ReconciliationProcessLog, error) {
	var logEntry model.ReconciliationProcessLog
	if err := d.db.First(&logEntry, logID).Error; err != nil {
		return logEntry, fmt.Errorf("log not found: %w", err)
	}
	return logEntry, nil
}

func (d *dao) UpdateReconciliationProcessLog(logEntry model.ReconciliationProcessLog) error {
	if err := d.db.Save(&logEntry).Error; err != nil {
		return fmt.Errorf("failed to update log: %w", err)
	}
	return nil
}
