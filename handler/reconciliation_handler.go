package handler

import (
	"encoding/json"
	"net/http"

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

func (h *ReconciliationHandler) ProcessReconciliation(w http.ResponseWriter, r *http.Request) {
	var req ProcessReconciliationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log, err := h.Usecase.ProcessReconciliation(req.TransactionCSVPath, req.ReferenceCSVPaths, req.StartTime, req.EndTime, req.Operator)
	if err != nil {
		http.Error(w, "Failed to process reconciliation", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(log)
}

func (h *ReconciliationHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	results, err := h.Usecase.GetReconciliationResults()
	if err != nil {
		http.Error(w, "Failed to get results", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}
