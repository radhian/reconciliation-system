package reconciliation

import "github.com/radhian/reconciliation_system/infra/db/model"

func (u *reconciliationUsecase) GetReconciliationResults() ([]model.ReconciliationProcessLog, error) {
	var logs []model.ReconciliationProcessLog
	if err := u.db.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
