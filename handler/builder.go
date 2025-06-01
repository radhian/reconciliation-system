package handler

import (
	usecase "github.com/radhian/reconciliation-system/usecase/reconciliation"
)

type ReconciliationHandler struct {
	Usecase usecase.ReconciliationUsecase
}

func NewReconciliationHandler(uc usecase.ReconciliationUsecase) *ReconciliationHandler {
	return &ReconciliationHandler{Usecase: uc}
}
