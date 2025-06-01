package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *ReconciliationHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	logIDStr := r.URL.Query().Get("log_id")
	if logIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "log_id is required",
		})
		return
	}

	logID, err := strconv.ParseInt(logIDStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "log_id must be a valid integer",
		})
		return
	}

	result, err := h.Usecase.GetReconciliationResult(logID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Failed to get result",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Status: "success",
		Data:   result,
	})
}
