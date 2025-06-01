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
