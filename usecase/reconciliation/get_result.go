package reconciliation

import (
	"github.com/radhian/reconciliation-system/infra/db/model"
)

func (u *reconciliationUsecase) GetReconciliationResults() ([]model.ReconciliationProcessLog, error) {
	return u.dao.GetReconciliationProcessLog()
}
