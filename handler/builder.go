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

type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
