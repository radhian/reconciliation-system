package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/radhian/reconciliation-system/entity"
)

func (h *ReconciliationHandler) ProcessReconciliation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req entity.ProcessReconciliationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
		return
	}

	startTime, endTime, err := parseAndConvertDates(req.StartDate, req.EndDate)
	if err != nil {
		log.Println("Invalid date input:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	if err := validateProcessReconciliationRequest(req); err != nil {
		log.Println("Invalid input:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	res, err := h.Usecase.ProcessReconciliationInit(
		req.TransactionCSVPath,
		req.ReferenceCSVPaths,
		startTime,
		endTime,
		req.Operator,
	)
	if err != nil {
		log.Printf("failed to load CSV: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Failed to process reconciliation",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Status: "success",
		Data:   res,
	})
}

func parseAndConvertDates(startDateStr, endDateStr string) (int64, int64, error) {
	const layout = "2006-01-02"

	startDate, err := time.Parse(layout, startDateStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start date format: %v", err)
	}
	startTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC).Unix()

	endDate, err := time.Parse(layout, endDateStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end date format: %v", err)
	}
	endTime := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC).Unix()

	if endTime < startTime {
		return 0, 0, errors.New("end date must not be before start date")
	}

	return startTime, endTime, nil
}

func validateProcessReconciliationRequest(req entity.ProcessReconciliationRequest) error {
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
			return fmt.Errorf("reference CSV file does not exist: %s", path)
		}
	}
	if strings.TrimSpace(req.StartDate) == "" || strings.TrimSpace(req.EndDate) == "" {
		return errors.New("start and end dates must be provided")
	}
	if strings.TrimSpace(req.Operator) == "" {
		return errors.New("operator must be specified")
	}
	return nil
}
