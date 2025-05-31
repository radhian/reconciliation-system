package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
)

func (h *ReconciliationHandler) ProcessReconciliation(w http.ResponseWriter, r *http.Request) {
	var req ProcessReconciliationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateProcessReconciliationRequest(req); err != nil {
		log.Println("Invalid input:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log, err := h.Usecase.ProcessReconciliationInit(req.TransactionCSVPath, req.ReferenceCSVPaths, req.StartTime, req.EndTime, req.Operator)
	if err != nil {
		http.Error(w, "Failed to process reconciliation", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(log)
}

func validateProcessReconciliationRequest(req ProcessReconciliationRequest) error {
	if req.TransactionCSVPath == "" {
		return errors.New("transaction CSV path is required")
	}
	if _, err := os.Stat(req.TransactionCSVPath); os.IsNotExist(err) {
		return errors.New("transaction CSV file does not exist")
	}
	if len(req.ReferenceCSVPaths) == 0 {
		return errors.New("at least one reference bank CSV path is required")
	}
	for _, path := range req.ReferenceCSVPaths {
		if path == "" {
			return errors.New("empty path found in reference CSV paths")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New("reference CSV file does not exist: " + path)
		}
	}
	if req.StartTime == 0 || req.EndTime == 0 {
		return errors.New("start and end times must be provided")
	}
	if req.EndTime < req.StartTime {
		return errors.New("end time must be after start time")
	}
	if strings.TrimSpace(req.Operator) == "" {
		return errors.New("operator must be specified")
	}
	return nil
}
