package handler

import (
	"encoding/json"
	"net/http"
)

func (h *ReconciliationHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	results, err := h.Usecase.GetReconciliationResults()
	if err != nil {
		http.Error(w, "Failed to get results", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}
