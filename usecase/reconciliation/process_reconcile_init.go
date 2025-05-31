package reconciliation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/radhian/reconciliation-system/infra/db/model"
)

type ProcessMetadata struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

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
	processInfo := ProcessMetadata{
		StartTime: startTime,
		EndTime:   endTime,
	}

	processInfoJSON, err := json.Marshal(processInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal process info: %w", err)
	}

	log := &model.ReconciliationProcessLog{
		ReconciliationType: 1, // set proper type if applicable
		TotalMainRow:       0, // will be updated after processing CSV
		CurrentMainRow:     0,
		ProcessInfo:        string(processInfoJSON),
		Status:             1, // init = 1, running = 2, finish = 3, cancel = 4
		Result:             "",
		CreateTime:         timeNowUnix,
		CreateBy:           operator,
		UpdateTime:         timeNowUnix,
		UpdateBy:           operator,
	}

	if err := u.db.Create(log).Error; err != nil {
		return nil, fmt.Errorf("failed to create reconciliation process log: %v", err)
	}

	for _, url := range append([]string{mainFileURL}, refFileURLs...) {
		asset := model.ReconciliationProcessLogAsset{
			ReconciliationProcessLogID: log.ID,
			FileName:                   filepath.Base(url),
			FileUrl:                    url,
			DataType:                   1, // 1 = main, 2 = reference
			CreateTime:                 timeNowUnix,
			CreateBy:                   operator,
		}
		if err := u.db.Create(&asset).Error; err != nil {
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
	destPath := filepath.Join("uploads", fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileName))

	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return "", err
	}

	return destPath, nil
}
