package model

type ReconciliationProcessLog struct {
	ID                 int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ReconciliationType int64  `gorm:"not null" json:"reconciliation_type"`
	TotalMainRow       int64  `gorm:"not null" json:"total_main_row"`
	CurrentMainRow     int64  `gorm:"not null" json:"current_main_row"`
	Status             int    `gorm:"not null" json:"status"`
	Result             string `gorm:"type:text;not null" json:"result"`
	CreateTime         int64  `gorm:"not null" json:"create_time"`
	CreateBy           string `gorm:"size:100;not null" json:"create_by"`
	UpdateTime         int64  `gorm:"not null" json:"update_time"`
	UpdateBy           string `gorm:"size:100;not null" json:"update_by"`
}
