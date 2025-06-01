package consts

const (
	// Reconciliation type bank transaction
	ReconciliationTypeBankTransaction = 1

	// Reconciliation status codes
	StatusInit     = 1
	StatusRunning  = 2
	StatusFinished = 3

	// DataType constants
	DataTypeSystemFile    = 1
	DataTypeBankStatement = 2

	// Default batch size for processing
	DefaultBatchSize = 1000
)
