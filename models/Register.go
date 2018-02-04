package models

type Register struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	Email      string `gorm:"unique;not null" json:"email"`
	Name       string `gorm:"column:fullName;not null" json:"full_name"`
	UserName   string `gorm:"column:userName;not null" json:"user_name"`
	UserId     uint32 `gorm:"column:userId;not null" json:"user_id"`
	IsPragyan  bool   `gorm:"column:isPragyan;not null" json:"is_pragyan"`
	Password   string `gorm:"column:pass;not null" json:"pass"`
	IsVerified bool   `gorm:"column:isVerified;not null" json:"is_verified"`
}

func (Register) TableName() string {
	return "Registers"
}
