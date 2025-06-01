package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *ReconciliationHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	logIDStr := r.URL.Query().Get("log_id")
	if logIDStr == "" {
		http.Error(w, "log_id is required", http.StatusBadRequest)
		return
	}

	logID, err := strconv.ParseInt(logIDStr, 10, 64)
	if err != nil {
		http.Error(w, "log_id must be a valid integer", http.StatusBadRequest)
		return
	}

	result, err := h.Usecase.GetReconciliationResult(logID)
	if err != nil {
		http.Error(w, "Failed to get result", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}
