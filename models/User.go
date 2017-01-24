package models

// User models the User object.
type User struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	Email     string `gorm:"unique;not null"`
	Name      string `gorm:"not null"`
	Cash      uint32 `gorm:"not null"`
	Total     uint32 `gorm:"not null"`
	CreatedAt string `gorm:"column:createdAt;not null"`
}

// User.TableName() is for letting Gorm know the correct table name.
func (User) TableName() string {
	return "Users"
}
