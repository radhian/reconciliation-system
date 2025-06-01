package reconciliation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/radhian/reconciliation-system/consts"
	"github.com/radhian/reconciliation-system/entity"
	"github.com/radhian/reconciliation-system/infra/db/model"
)

func (u *reconciliationUsecase) ProcessReconciliationInit(transactionCSV string, referenceCSVs []string, startTime, endTime int64, operator string) (*model.ReconciliationProcessLog, error) {
	timeNowUnix := time.Now().Unix()

	mainFileURL, err := u.uploadFile(transactionCSV)
	if err != nil {
		return nil, fmt.Errorf("failed to upload main file: %v", err)
	}

	refFileURLs := make([]string, 0, len(referenceCSVs))
	for _, ref := range referenceCSVs {
		url, err := u.uploadFile(ref)
		if err != nil {
			return nil, fmt.Errorf("failed to upload reference file %s: %v", ref, err)
		}
		refFileURLs = append(refFileURLs, url)
	}

	// Create process info
	processInfo := entity.ProcessMetadata{
		StartTime: startTime,
		EndTime:   endTime,
	}

	processInfoJSON, err := json.Marshal(processInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal process info: %w", err)
	}

	log := &model.ReconciliationProcessLog{
		ReconciliationType: consts.ReconciliationTypeBankTransaction,
		TotalMainRow:       0, // will be updated after processing CSV
		CurrentMainRow:     0,
		ProcessInfo:        string(processInfoJSON),
		Status:             consts.StatusInit,
		Result:             "",
		CreateTime:         timeNowUnix,
		CreateBy:           operator,
		UpdateTime:         timeNowUnix,
		UpdateBy:           operator,
	}

	if err := u.dao.CreateReconciliationProcessLog(log); err != nil {
		return nil, fmt.Errorf("failed to create reconciliation process log: %v", err)
	}

	for i, url := range append([]string{mainFileURL}, refFileURLs...) {
		dataType := int64(consts.DataTypeSystemFile)
		if i > 0 {
			dataType = consts.DataTypeBankStatement
		}

		asset := &model.ReconciliationProcessLogAsset{
			ReconciliationProcessLogID: log.ID,
			FileName:                   filepath.Base(url),
			FileUrl:                    url,
			DataType:                   dataType,
			CreateTime:                 timeNowUnix,
			CreateBy:                   operator,
		}
		if err := u.dao.CreateReconciliationProcessLogAsset(asset); err != nil {
			return nil, fmt.Errorf("failed to save file asset: %v", err)
		}
	}

	return log, nil
}

// NOTES: this is the simulation version of object storage, later we can implement object storage uploader in production.
func (u *reconciliationUsecase) uploadFile(filePath string) (string, error) {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	fileName := filepath.Base(filePath)

	uploadsDir := "uploads"

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	destPath := filepath.Join(uploadsDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileName))

	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return "", err
	}

	return destPath, nil
}
