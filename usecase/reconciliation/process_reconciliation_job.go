package reconciliation

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/radhian/reconciliation-system/infra/db/model"
)

const (
	// Reconciliation status codes
	StatusRunning  = 2
	StatusFinished = 3

	// DataType constants
	DataTypeSystemFile    = 1
	DataTypeBankStatement = 2

	// Default batch size for processing
	DefaultBatchSize = 1000
)

type Transaction struct {
	TrxID           string
	Amount          float64
	Type            string // DEBIT or CREDIT
	TransactionTime time.Time
}

type BankStatement struct {
	BankName         string
	UniqueIdentifier string
	Amount           float64
	Date             time.Time
}

func (u *reconciliationUsecase) ProcessReconciliationJob(ctx context.Context, logID int64) error {
	logEntry, err := u.fetchProcessLog(logID)
	if err != nil {
		return err
	}

	assets, err := u.fetchProcessLogAssets(logID)
	if err != nil {
		return err
	}

	systemFileUrl, err := findSystemFileUrl(assets)
	if err != nil {
		return err
	}

	requestStartTime, requestEndTime, err := parseProcessMetadata(logEntry.ProcessInfo)
	if err != nil {
		return err
	}

	totalRows, processedRows, result := u.reconcileData(
		systemFileUrl,
		assets,
		requestStartTime,
		requestEndTime,
		int(logEntry.CurrentMainRow),
		DefaultBatchSize,
	)

	u.updateProcessLogAfterBatch(logEntry, totalRows, processedRows, result, requestStartTime, requestEndTime)

	if err := u.dao.UpdateReconciliationProcessLog(logEntry).Error; err != nil {
		return fmt.Errorf("failed to update log: %w", err)
	}

	return nil
}

func (u *reconciliationUsecase) fetchProcessLog(logID int64) (model.ReconciliationProcessLog, error) {
	var logEntry model.ReconciliationProcessLog
	logEntry, err := u.dao.GetReconciliationProcessLogByID(uint(logID))
	if err != nil {
		return logEntry, fmt.Errorf("log not found: %w", err)
	}
	return logEntry, nil
}

func (u *reconciliationUsecase) fetchProcessLogAssets(logID int64) ([]model.ReconciliationProcessLogAsset, error) {
	var assets []model.ReconciliationProcessLogAsset
	assets, err := u.dao.GetReconciliationLogAssetsByLogID(uint(logID))
	if err != nil {
		return nil, fmt.Errorf("log not found: %w", err)
	}
	return assets, nil
}

func findSystemFileUrl(assets []model.ReconciliationProcessLogAsset) (string, error) {
	for _, asset := range assets {
		if asset.DataType == DataTypeSystemFile {
			return asset.FileUrl, nil
		}
	}
	return "", errors.New("missing system file URL")
}

func parseProcessMetadata(processInfo string) (time.Time, time.Time, error) {
	var metadata ProcessMetadata
	if err := json.Unmarshal([]byte(processInfo), &metadata); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse process metadata: %w", err)
	}
	start := time.Unix(metadata.StartTime, 0)
	end := time.Unix(metadata.EndTime, 0)
	return start, end, nil
}

func (u *reconciliationUsecase) updateProcessLogAfterBatch(
	logEntry model.ReconciliationProcessLog,
	totalRows int64,
	processedRows int64,
	result string,
	requestStartTime time.Time,
	requestEndTime time.Time,
) {
	logEntry.TotalMainRow = totalRows
	logEntry.CurrentMainRow += processedRows
	logEntry.Result = result

	if logEntry.CurrentMainRow >= totalRows {
		logEntry.Status = StatusFinished
	} else {
		logEntry.Status = StatusRunning
	}

	logEntry.UpdateTime = time.Now().Unix()
	logEntry.UpdateBy = "system"
}

func (u *reconciliationUsecase) reconcileData(
	systemFileUrl string,
	assets []model.ReconciliationProcessLogAsset,
	startTime time.Time,
	endTime time.Time,
	startIndex int,
	batchSize int,
) (totalRows int64, processedRows int64, result string) {
	systemTxsAll, err := parseSystemTransactions(systemFileUrl, startTime, endTime)
	if err != nil {
		log.Errorf("failed to parse system transactions: %v", err)
		return 0, 0, "failed"
	}
	totalSystemRows := len(systemTxsAll)

	if startIndex < 0 || startIndex >= totalSystemRows {
		return int64(totalSystemRows), 0, "{}"
	}

	endIndex := startIndex + batchSize
	if endIndex > totalSystemRows {
		endIndex = totalSystemRows
	}
	systemTxsBatch := systemTxsAll[startIndex:endIndex]

	bankTxs, bankBySource := u.parseBankAssets(assets, startTime, endTime)

	sysMap := buildTransactionMap(systemTxsBatch)
	bankMap := buildBankStatementMap(bankTxs)

	matchedCount, unmatchedSys, unmatchedBank := compareTransactions(sysMap, bankMap)
	bankGroups := groupUnmatchedBanks(unmatchedBank, bankBySource)

	resultSummary, err := buildResultSummary(len(systemTxsBatch), int(matchedCount), unmatchedSys, bankGroups)
	if err != nil {
		log.Errorf("failed to build result summary: %v", err)
		return int64(totalSystemRows), int64(len(systemTxsBatch)), "{}"
	}

	return int64(totalSystemRows), int64(len(systemTxsBatch)), resultSummary
}

func (u *reconciliationUsecase) parseBankAssets(
	assets []model.ReconciliationProcessLogAsset,
	startTime, endTime time.Time,
) ([]BankStatement, map[string][]BankStatement) {
	bankTxs := make([]BankStatement, 0)
	bankBySource := make(map[string][]BankStatement)

	for _, asset := range assets {
		if asset.DataType != DataTypeBankStatement {
			continue
		}
		txs, err := parseBankStatements(asset.FileUrl, startTime, endTime)
		if err != nil {
			log.Errorf("failed to parse bank statements from %s: %v", asset.FileUrl, err)
			continue
		}
		bankTxs = append(bankTxs, txs...)
		bankBySource[asset.FileName] = append(bankBySource[asset.FileName], txs...)
	}

	return bankTxs, bankBySource
}

func buildTransactionMap(transactions []Transaction) map[string]Transaction {
	m := make(map[string]Transaction)
	for _, tx := range transactions {
		key := fmt.Sprintf("%s|%.2f", tx.TrxID, tx.Amount)
		m[key] = tx
	}
	return m
}

func buildBankStatementMap(bankTxs []BankStatement) map[string]BankStatement {
	m := make(map[string]BankStatement)
	for _, b := range bankTxs {
		key := fmt.Sprintf("%s|%.2f", b.UniqueIdentifier, b.Amount)
		m[key] = b
	}
	return m
}

func compareTransactions(
	sysMap map[string]Transaction,
	bankMap map[string]BankStatement,
) (matchedCount int64, unmatchedSys []Transaction, unmatchedBank []BankStatement) {
	for key, tx := range sysMap {
		if _, found := bankMap[key]; found {
			matchedCount++
		} else {
			unmatchedSys = append(unmatchedSys, tx)
		}
	}

	for key, b := range bankMap {
		if _, found := sysMap[key]; !found {
			unmatchedBank = append(unmatchedBank, b)
		}
	}

	return matchedCount, unmatchedSys, unmatchedBank
}

func groupUnmatchedBanks(
	unmatchedBank []BankStatement,
	bankBySource map[string][]BankStatement,
) map[string][]BankStatement {
	bankGroups := make(map[string][]BankStatement)

	for _, b := range unmatchedBank {
		for source, list := range bankBySource {
			for _, ref := range list {
				if ref.UniqueIdentifier == b.UniqueIdentifier && ref.Amount == b.Amount {
					bankGroups[source] = append(bankGroups[source], b)
					break
				}
			}
		}
	}
	return bankGroups
}

func buildResultSummary(
	total int,
	matched int,
	unmatchedSystem []Transaction,
	unmatchedBank map[string][]BankStatement,
) (string, error) {
	summary := struct {
		TotalProcessed     int                        `json:"total_processed"`
		Matched            int                        `json:"matched"`
		Unmatched          int                        `json:"unmatched"`
		SystemUnmatched    []Transaction              `json:"system_unmatched"`
		BankUnmatchedBySrc map[string][]BankStatement `json:"bank_unmatched_by_source"`
	}{
		TotalProcessed:     total,
		Matched:            matched,
		Unmatched:          total - matched,
		SystemUnmatched:    unmatchedSystem,
		BankUnmatchedBySrc: unmatchedBank,
	}

	resBytes, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(resBytes), nil
}

func parseSystemTransactions(sourceFile string, startTime, endTime time.Time) ([]Transaction, error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open system file %s: %w", sourceFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV from system file %s: %w", sourceFile, err)
	}

	var transactions []Transaction
	for i, record := range records {
		if i == 0 {
			// Assuming first row is header, skip it
			continue
		}
		if len(record) < 4 {
			// Invalid row
			continue
		}

		trxID := record[0]

		amount, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue // skip malformed amount
		}

		txType := record[2]

		txTime, err := time.Parse(time.RFC3339, record[3])
		if err != nil {
			continue // skip malformed time
		}

		if txTime.Before(startTime) || txTime.After(endTime) {
			continue // out of time range
		}

		transaction := Transaction{
			TrxID:           trxID,
			Amount:          amount,
			Type:            txType,
			TransactionTime: txTime,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func parseBankStatements(sourceFile string, startTime, endTime time.Time) ([]BankStatement, error) {
	file, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open bank statement file %s: %w", sourceFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV from bank statement file %s: %w", sourceFile, err)
	}

	var statements []BankStatement
	for i, record := range records {
		if i == 0 {
			// Assuming first row is header, skip it
			continue
		}
		if len(record) < 4 {
			// Invalid row
			continue
		}

		bankName := record[0]
		uniqueID := record[1]

		amount, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue // skip malformed amount
		}

		date, err := time.Parse(time.RFC3339, record[3])
		if err != nil {
			continue // skip malformed date
		}

		if date.Before(startTime) || date.After(endTime) {
			continue // out of time range
		}

		statement := BankStatement{
			BankName:         bankName,
			UniqueIdentifier: uniqueID,
			Amount:           amount,
			Date:             date,
		}

		statements = append(statements, statement)
	}

	return statements, nil
}
