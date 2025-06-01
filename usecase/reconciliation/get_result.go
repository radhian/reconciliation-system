package reconciliation

import (
	"github.com/radhian/reconciliation-system/infra/db/model"
)

func (u *reconciliationUsecase) GetReconciliationResult(logID int64) (model.ReconciliationProcessLog, error) {
	return u.dao.GetReconciliationProcessLogByID(uint(logID))
}
