package reconciliation

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
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
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("[ReconcileJob] Recovered from panic for LogID %d: %v", logID, r)
		}
	}()

	log.Infof("[ReconcileJob] Starting reconciliation job for LogID: %d", logID)

	logEntry, err := u.fetchProcessLog(logID)
	if err != nil {
		log.Errorf("[ReconcileJob] Failed to fetch process log for LogID %d: %v", logID, err)
		return err
	}
	log.Infof("[ReconcileJob] Fetched process log: %+v", logEntry)

	assets, err := u.fetchProcessLogAssets(logID)
	if err != nil {
		log.Errorf("[ReconcileJob] Failed to fetch assets for LogID %d: %v", logID, err)
		return err
	}
	log.Infof("[ReconcileJob] Fetched assets: %+v", assets)

	systemFileUrl, err := findSystemFileUrl(assets)
	if err != nil {
		log.Errorf("[ReconcileJob] System file URL not found in assets: %v", err)
		return err
	}
	log.Infof("[ReconcileJob] Found system file URL: %s", systemFileUrl)

	requestStartTime, requestEndTime, err := parseProcessMetadata(logEntry.ProcessInfo)
	if err != nil {
		log.Errorf("[ReconcileJob] Failed to parse metadata for LogID %d: %v", logID, err)
		return err
	}
	log.Infof("[ReconcileJob] Parsed metadata -> start: %s, end: %s", requestStartTime, requestEndTime)

	log.Infof("[ReconcileJob] Reconciling data batch from row %d with batch size %d", logEntry.CurrentMainRow, DefaultBatchSize)

	totalRows, processedRows, result := u.reconcileData(
		systemFileUrl,
		assets,
		requestStartTime,
		requestEndTime,
		int(logEntry.CurrentMainRow),
		DefaultBatchSize,
	)

	log.Infof("[ReconcileJob] Reconciliation result for LogID %d -> totalRows: %d, processedRows: %d", logID, totalRows, processedRows)
	log.Infof("[ReconcileJob] Reconciliation summary: %s", result)

	logEntry = u.updateProcessLogAfterBatch(logEntry, totalRows, processedRows, result, requestStartTime, requestEndTime)

	if err := u.dao.UpdateReconciliationProcessLog(logEntry); err != nil {
		log.Errorf("[ReconcileJob] Failed to update process log for LogID %d: %v", logID, err)
		return fmt.Errorf("failed to update log: %w", err)
	}

	log.Infof("[ReconcileJob] Successfully finished processing batch for LogID %d", logID)
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
	start := time.Unix(metadata.StartTime, 0).UTC()
	end := time.Unix(metadata.EndTime, 0).UTC()
	return start, end, nil
}

func (u *reconciliationUsecase) updateProcessLogAfterBatch(
	logEntry model.ReconciliationProcessLog,
	totalRows int64,
	processedRows int64,
	result string,
	requestStartTime time.Time,
	requestEndTime time.Time,
) model.ReconciliationProcessLog {
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

	return logEntry
}

func (u *reconciliationUsecase) reconcileData(
	systemFileUrl string,
	assets []model.ReconciliationProcessLogAsset,
	startTime time.Time,
	endTime time.Time,
	startIndex int,
	batchSize int,
) (totalRows int64, processedRows int64, result string) {
	log.Infof("[Reconcile] Starting reconciliation for file: %s", systemFileUrl)

	systemTxsAll, err := parseSystemTransactions(systemFileUrl, startTime, endTime)
	if err != nil {
		log.Infof("[Reconcile] Failed to parse system transactions: %v", err)
		return 0, 0, "failed"
	}
	log.Infof("[Reconcile] Total system transactions in time range: %d", len(systemTxsAll))
	totalSystemRows := len(systemTxsAll)

	if startIndex < 0 || startIndex >= totalSystemRows {
		log.Infof("[Reconcile] Start index %d out of bounds (total %d)", startIndex, totalSystemRows)
		return int64(totalSystemRows), 0, "{}"
	}

	endIndex := startIndex + batchSize
	if endIndex > totalSystemRows {
		endIndex = totalSystemRows
	}
	systemTxsBatch := systemTxsAll[startIndex:endIndex]
	log.Infof("[Reconcile] Processing transactions from index %d to %d", startIndex, endIndex)

	bankTxs, bankBySource := u.parseBankAssets(assets, startTime, endTime)
	log.Infof("[Reconcile] Total parsed bank transactions: %d", len(bankTxs))

	sysMap := buildTransactionMap(systemTxsBatch)
	bankMap := buildBankStatementMap(bankTxs)

	matchedCount, unmatchedSys, unmatchedBank := compareTransactions(sysMap, bankMap)
	log.Infof("[Reconcile] Matched: %d, Unmatched System: %d, Unmatched Bank: %d", matchedCount, len(unmatchedSys), len(unmatchedBank))

	bankGroups := groupUnmatchedBanks(unmatchedBank, bankBySource)

	resultSummary, err := buildResultSummary(len(systemTxsBatch), int(matchedCount), unmatchedSys, bankGroups)
	if err != nil {
		log.Infof("[Reconcile] Failed to build result summary: %v", err)
		return int64(totalSystemRows), int64(len(systemTxsBatch)), "{}"
	}

	log.Infof("[Reconcile] Successfully built result summary")
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
	for _, trx := range transactions {
		var typeCode string
		if trx.Type == "CREDIT" {
			typeCode = "c"
		} else if trx.Type == "DEBIT" {
			typeCode = "d"
		} else {
			typeCode = "u"
		}

		key := fmt.Sprintf("%s|%.2f", typeCode, trx.Amount)
		m[key] = trx
	}
	return m
}

func buildBankStatementMap(bankTxs []BankStatement) map[string]BankStatement {
	m := make(map[string]BankStatement)
	for _, b := range bankTxs {
		var typeCode string
		if b.Amount < 0 {
			typeCode = "d"
		} else {
			typeCode = "c"
		}

		amount := math.Abs(b.Amount)
		key := fmt.Sprintf("%s|%.2f", typeCode, amount)
		m[key] = b
	}
	return m
}

func compareTransactions(
	sysMap map[string]Transaction,
	bankMap map[string]BankStatement,
) (matchedCount int64, unmatchedSys []Transaction, unmatchedBank []BankStatement) {
	for key, sysTx := range sysMap {
		log.Printf("123key:%s bankMap:%v", key, bankMap)
		if _, found := bankMap[key]; found {
			matchedCount++
			delete(bankMap, key) // Optional: so we don't re-count it later
		} else {
			unmatchedSys = append(unmatchedSys, sysTx)
		}
	}

	for _, bankTx := range bankMap {
		unmatchedBank = append(unmatchedBank, bankTx)
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
	log.Infof("[SystemParser] Reading system file: %s", sourceFile)

	file, err := os.Open(sourceFile)
	if err != nil {
		log.Infof("[SystemParser] Failed to open file: %v", err)
		return nil, fmt.Errorf("failed to open system file %s: %w", sourceFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Infof("[SystemParser] Failed to read CSV: %v", err)
		return nil, fmt.Errorf("failed to read CSV from system file %s: %w", sourceFile, err)
	}

	var transactions []Transaction
	for i, record := range records {
		if i == 0 {
			continue
		}

		trxID := strings.TrimSpace(record[0])
		if trxID == "" || len(record) < 4 {
			log.Infof("[SystemParser1] Skipping row %d due to parse error: %v", i, err)
			continue
		}

		amount, err := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		if err != nil {
			log.Infof("[SystemParser2] Skipping row %d due to parse error: %v", i, err)
			continue
		}

		txType := strings.ToUpper(strings.TrimSpace(record[2])) // Optional: normalize casing
		txTimeRaw := strings.TrimSpace(record[3])
		txTime, err := time.Parse(time.RFC3339, txTimeRaw)
		log.Printf("txTime:%v startTime:%v, endTime:%v", txTime.String(), startTime.String(), endTime.String())
		if err != nil || txTime.Before(startTime) || txTime.After(endTime) {
			log.Infof("[SystemParser3] Skipping row %d due to parse error: %v", i, err)
			continue
		}

		transactions = append(transactions, Transaction{
			TrxID:           trxID,
			Amount:          amount,
			Type:            txType,
			TransactionTime: txTime,
		})
	}

	log.Infof("[SystemParser] Parsed %d valid system transactions", len(transactions))
	return transactions, nil
}

func parseBankStatements(sourceFile string, startTime, endTime time.Time) ([]BankStatement, error) {
	log.Infof("[BankParser] Reading bank statement file: %s", sourceFile)

	file, err := os.Open(sourceFile)
	if err != nil {
		log.Infof("[BankParser] Failed to open file: %v", err)
		return nil, fmt.Errorf("failed to open bank statement file %s: %w", sourceFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Infof("[BankParser] Failed to read CSV: %v", err)
		return nil, fmt.Errorf("failed to read CSV from bank statement file %s: %w", sourceFile, err)
	}

	// Truncate start and end time to date only
	startDate := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	endDate := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, endTime.Location())

	var statements []BankStatement
	for i, record := range records {
		if i == 0 {
			continue // skip header
		}
		if len(record) < 3 {
			log.Infof("[BankParser] Skipping row %d: insufficient fields (%v)", i, record)
			continue
		}

		amount, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			log.Infof("[BankParser] Skipping row %d: invalid amount '%s'", i, record[1])
			continue
		}

		date, err := time.Parse("2006-01-02", record[2])
		if err != nil {
			log.Infof("[BankParser] Skipping row %d: invalid date format '%s'", i, record[2])
			continue
		}

		// Truncate parsed date to just the date
		dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

		log.Infof("[BankParser] Row %d date check: date=%s | startDate=%s | endDate=%s",
			i,
			dateOnly.Format("2006-01-02"),
			startDate.Format("2006-01-02"),
			endDate.Format("2006-01-02"),
		)

		if dateOnly.Before(startDate) || dateOnly.After(endDate) {
			log.Infof("[BankParser] Skipping row %d: date out of range", i)
			continue
		}

		statements = append(statements, BankStatement{
			BankName:         "Unknown", // optional: adjust as needed
			UniqueIdentifier: record[0],
			Amount:           amount,
			Date:             dateOnly,
		})
	}

	log.Infof("[BankParser] Parsed %d valid bank statements", len(statements))
	return statements, nil
}
