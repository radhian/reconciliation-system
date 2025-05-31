package reconciliation

import "github.com/radhian/reconciliation_system/infra/db/model"

func (u *reconciliationUsecase) ProcessReconciliation(transactionCSV string, referenceCSVs []string, startTime, endTime int64, operator string) (*model.ReconciliationProcessLog, error) {
	log := model.ReconciliationProcessLog{
		ReconciliationType: 1,
		TotalMainRow:       100,
		CurrentMainRow:     0,
		Status:             0,
		Result:             "{}",
		CreateTime:         startTime,
		CreateBy:           operator,
		UpdateTime:         endTime,
		UpdateBy:           operator,
	}
	if err := u.db.Create(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}
