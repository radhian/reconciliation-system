package dao

import (
	"fmt"

	"github.com/radhian/reconciliation-system/infra/db/model"
)

func (d *dao) CreateReconciliationProcessLogAsset(payload model.ReconciliationProcessLogAsset) error {
	if err := d.db.Create(&payload).Error; err != nil {
		return fmt.Errorf("failed to save file asset: %v", err)
	}
	return nil
}

func (d *dao) GetReconciliationLogAssetsByLogID(logID uint) ([]model.ReconciliationProcessLogAsset, error) {
	var assets []model.ReconciliationProcessLogAsset
	if err := d.db.Where("reconciliation_process_log_id = ?", logID).Find(&assets).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch log assets: %w", err)
	}
	return assets, nil
}
