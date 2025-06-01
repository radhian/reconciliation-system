package entity

import "time"

type Transaction struct {
	TrxID           string
	Amount          float64
	Type            string // DEBIT or CREDIT
	TransactionTime time.Time
}

type BankStatement struct {
	UniqueIdentifier string
	Amount           float64
	Date             time.Time
}

type ProcessReconciliationRequest struct {
	TransactionCSVPath string   `json:"transaction_csv_path"`
	ReferenceCSVPaths  []string `json:"reference_csv_paths"`
	StartDate          string   `json:"start_date"`
	EndDate            string   `json:"end_date"`
	Operator           string   `json:"operator"`
}

type ProcessMetadata struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}
