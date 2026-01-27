package model

type IPBypass struct {
	ID        int64  `gorm:"column:id;primaryKey;autoIncrement"`
	CIDR      string `gorm:"column:cidr"`
	Domain    string `gorm:"column:domain"`
	ExpiresAt int64  `gorm:"column:expires_at"`
	Note      string `gorm:"column:note"`
	CreatedAt int64  `gorm:"column:created_at"`
}

func (IPBypass) TableName() string {
	return "ip_bypasses"
}
