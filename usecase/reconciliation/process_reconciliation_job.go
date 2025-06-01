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
	"github.com/radhian/reconciliation-system/consts"
	"github.com/radhian/reconciliation-system/entity"
	"github.com/radhian/reconciliation-system/infra/db/model"
	"github.com/radhian/reconciliation-system/utils"
)

func (u *reconciliationUsecase) ProcessReconciliationJob(ctx context.Context, logID int64) error {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("[ReconcileJob] Panic recovered for LogID %d: %v", logID, r)
		}
	}()

	log.Infof("[ReconcileJob] Starting job for LogID: %d", logID)

	logEntry, err := u.fetchProcessLog(logID)
	if err != nil {
		log.Errorf("[ReconcileJob] Could not fetch process log %d: %v", logID, err)
		return err
	}

	assets, err := u.fetchProcessLogAssets(logID)
	if err != nil {
		log.Errorf("[ReconcileJob] Could not fetch assets for LogID %d: %v", logID, err)
		return err
	}

	systemFileUrl, err := findSystemFileUrl(assets)
	if err != nil {
		log.Errorf("[ReconcileJob] System file URL not found: %v", err)
		return err
	}

	requestStartTime, requestEndTime, err := parseProcessMetadata(logEntry.ProcessInfo)
	if err != nil {
		log.Errorf("[ReconcileJob] Metadata parse error for LogID %d: %v", logID, err)
		return err
	}

	log.Infof("[ReconcileJob] Reconciling batch (start row: %d, size: %d)", logEntry.CurrentMainRow, u.batchSize)

	totalRows, processedRows, result := u.reconcileData(
		systemFileUrl,
		assets,
		requestStartTime,
		requestEndTime,
		int(logEntry.CurrentMainRow),
		int(u.batchSize),
	)

	log.Infof("[ReconcileJob] Batch done for LogID %d: total=%d, processed=%d", logID, totalRows, processedRows)

	logEntry = u.updateProcessLogAfterBatch(logEntry, totalRows, processedRows, result, requestStartTime, requestEndTime)

	if err := u.dao.UpdateReconciliationProcessLog(logEntry); err != nil {
		log.Errorf("[ReconcileJob] Failed to update log %d: %v", logID, err)
		return fmt.Errorf("failed to update log: %w", err)
	}

	log.Infof("[ReconcileJob] Job completed for LogID %d", logID)
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
		if asset.DataType == consts.DataTypeSystemFile {
			return asset.FileUrl, nil
		}
	}
	return "", errors.New("missing system file URL")
}

func parseProcessMetadata(processInfo string) (time.Time, time.Time, error) {
	var metadata entity.ProcessMetadata
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
		logEntry.Status = consts.StatusFinished
	} else {
		logEntry.Status = consts.StatusRunning
	}

	logEntry.UpdateTime = time.Now().Unix()
	logEntry.UpdateBy = "system"

	return logEntry
}

func (u *reconciliationUsecase) parseBankAssets(
	assets []model.ReconciliationProcessLogAsset,
	startTime, endTime time.Time,
) ([]entity.BankStatement, map[string][]entity.BankStatement) {
	bankTxs := make([]entity.BankStatement, 0)
	bankBySource := make(map[string][]entity.BankStatement)

	for _, asset := range assets {
		if asset.DataType != consts.DataTypeBankStatement {
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

func buildTransactionMap(transactions []entity.Transaction) map[string][]entity.Transaction {
	m := make(map[string][]entity.Transaction)
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
		m[key] = append(m[key], trx)
	}
	return m
}

func buildBankStatementMap(bankTxs []entity.BankStatement) map[string][]entity.BankStatement {
	m := make(map[string][]entity.BankStatement)
	for _, b := range bankTxs {
		var typeCode string
		if b.Amount < 0 {
			typeCode = "d"
		} else {
			typeCode = "c"
		}

		amount := math.Abs(b.Amount)
		key := fmt.Sprintf("%s|%.2f", typeCode, amount)
		m[key] = append(m[key], b)
	}
	return m
}

func compareTransactions(
	sysMap map[string][]entity.Transaction,
	bankMap map[string][]entity.BankStatement,
) (matchedCount int64, unmatchedSys []entity.Transaction, unmatchedBank []entity.BankStatement) {
	for key, txList := range sysMap {
		bankList, found := bankMap[key]
		if found {
			matchCount := utils.Min(len(txList), len(bankList))
			matchedCount += int64(matchCount)

			if len(txList) > matchCount {
				unmatchedSys = append(unmatchedSys, txList[matchCount:]...)
			}
			if len(bankList) > matchCount {
				unmatchedBank = append(unmatchedBank, bankList[matchCount:]...)
			}
			delete(bankMap, key)
		} else {
			unmatchedSys = append(unmatchedSys, txList...)
		}
	}

	for _, leftover := range bankMap {
		unmatchedBank = append(unmatchedBank, leftover...)
	}

	return matchedCount, unmatchedSys, unmatchedBank
}

func groupUnmatchedBanks(
	unmatchedBank []entity.BankStatement,
	bankBySource map[string][]entity.BankStatement,
) map[string][]entity.BankStatement {
	bankGroups := make(map[string][]entity.BankStatement)

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
	unmatchedSystem []entity.Transaction,
	unmatchedBank map[string][]entity.BankStatement,
) (string, error) {
	var totalDiscrepancy float64

	// Sum discrepancies from system unmatched
	for _, trx := range unmatchedSystem {
		totalDiscrepancy += trx.Amount
	}

	// Sum discrepancies from bank unmatched
	for _, group := range unmatchedBank {
		for _, b := range group {
			totalDiscrepancy += math.Abs(b.Amount)
		}
	}

	summary := struct {
		TotalProcessed     int                               `json:"total_processed"`
		Matched            int                               `json:"matched"`
		Unmatched          int                               `json:"unmatched"`
		SystemUnmatched    []entity.Transaction              `json:"system_unmatched"`
		BankUnmatchedBySrc map[string][]entity.BankStatement `json:"bank_unmatched_by_source"`
		TotalDiscrepancy   float64                           `json:"total_discrepancy"`
	}{
		TotalProcessed:     total,
		Matched:            matched,
		Unmatched:          total - matched,
		SystemUnmatched:    unmatchedSystem,
		BankUnmatchedBySrc: unmatchedBank,
		TotalDiscrepancy:   totalDiscrepancy,
	}

	resBytes, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(resBytes), nil
}

func (u *reconciliationUsecase) reconcileData(
	systemFileUrl string,
	assets []model.ReconciliationProcessLogAsset,
	startTime time.Time,
	endTime time.Time,
	startIndex int,
	batchSize int,
) (totalRows int64, processedRows int64, result string) {
	log.Infof("[Reconcile] Start file: %s", systemFileUrl)

	systemTxsAll, err := parseSystemTransactions(systemFileUrl, startTime, endTime)
	if err != nil {
		log.Errorf("[Reconcile] System parse failed: %v", err)
		return 0, 0, "failed"
	}
	totalSystemRows := len(systemTxsAll)
	log.Infof("[Reconcile] Found %d system transactions in range", totalSystemRows)

	if startIndex < 0 || startIndex >= totalSystemRows {
		log.Warnf("[Reconcile] Invalid start index %d of %d", startIndex, totalSystemRows)
		return int64(totalSystemRows), 0, "{}"
	}

	endIndex := startIndex + batchSize
	if endIndex > totalSystemRows {
		endIndex = totalSystemRows
	}
	systemTxsBatch := systemTxsAll[startIndex:endIndex]

	bankTxs, bankBySource := u.parseBankAssets(assets, startTime, endTime)
	log.Infof("[Reconcile] Parsed %d bank transactions", len(bankTxs))

	sysMap := buildTransactionMap(systemTxsBatch)
	bankMap := buildBankStatementMap(bankTxs)

	matchedCount, unmatchedSys, unmatchedBank := compareTransactions(sysMap, bankMap)
	log.Infof("[Reconcile] Matched: %d | Unmatched: System=%d, Bank=%d",
		matchedCount, len(unmatchedSys), len(unmatchedBank))

	bankGroups := groupUnmatchedBanks(unmatchedBank, bankBySource)

	resultSummary, err := buildResultSummary(len(systemTxsBatch), int(matchedCount), unmatchedSys, bankGroups)
	if err != nil {
		log.Errorf("[Reconcile] Failed to build result: %v", err)
		return int64(totalSystemRows), int64(len(systemTxsBatch)), "{}"
	}

	return int64(totalSystemRows), int64(len(systemTxsBatch)), resultSummary
}

func parseSystemTransactions(sourceFile string, startTime, endTime time.Time) ([]entity.Transaction, error) {
	log.Infof("[SystemParser] Reading system file: %s", sourceFile)

	file, err := os.Open(sourceFile)
	if err != nil {
		log.Errorf("[SystemParser] Failed to open file: %v", err)
		return nil, fmt.Errorf("failed to open system file %s: %w", sourceFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Errorf("[SystemParser] Failed to read CSV: %v", err)
		return nil, fmt.Errorf("failed to read CSV from system file %s: %w", sourceFile, err)
	}

	var transactions []entity.Transaction
	skipped := 0

	for i, record := range records {
		if i == 0 || len(record) < 4 || strings.TrimSpace(record[0]) == "" {
			skipped++
			continue
		}

		amount, err1 := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		txTime, err2 := time.Parse(time.RFC3339, strings.TrimSpace(record[3]))
		if err1 != nil || err2 != nil || txTime.Before(startTime) || txTime.After(endTime) {
			skipped++
			continue
		}

		transactions = append(transactions, entity.Transaction{
			TrxID:           strings.TrimSpace(record[0]),
			Amount:          amount,
			Type:            strings.ToUpper(strings.TrimSpace(record[2])),
			TransactionTime: txTime,
		})
	}

	log.Infof("[SystemParser] Parsed %d transactions, skipped %d invalid rows", len(transactions), skipped)
	return transactions, nil
}

func parseBankStatements(sourceFile string, startTime, endTime time.Time) ([]entity.BankStatement, error) {
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

	var statements []entity.BankStatement
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

		statements = append(statements, entity.BankStatement{
			UniqueIdentifier: record[0],
			Amount:           amount,
			Date:             dateOnly,
		})
	}

	log.Infof("[BankParser] Parsed %d valid bank statements", len(statements))
	return statements, nil
}
