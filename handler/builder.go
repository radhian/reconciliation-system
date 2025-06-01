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

type ProcessReconciliationRequest struct {
	TransactionCSVPath string   `json:"transaction_csv_path"`
	ReferenceCSVPaths  []string `json:"reference_csv_paths"`
	StartTime          int64    `json:"start_time"`
	EndTime            int64    `json:"end_time"`
	Operator           string   `json:"operator"`
}
