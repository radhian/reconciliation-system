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

	// Default config
	DefaultBatchSize     = 1000
	DefaultWorkerNumber  = 1
	DefaultIntervalInSec = 2
)
