package model

type ReconciliationProcessLogAsset struct {
	ID                         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ReconciliationProcessLogID int64  `gorm:"not null;index" json:"reconciliation_process_log_id"`
	DataType                   int64  `gorm:"not null" json:"data_type"`
	FileName                   string `gorm:"size:100;not null" json:"file_name"`
	FileUrl                    string `gorm:"size:100;not null" json:"file_url"`
	CreateTime                 int64  `gorm:"not null" json:"create_time"`
	CreateBy                   string `gorm:"size:100;not null" json:"create_by"`
}
