package TbFolder

type Folder struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"type:varchar(255)" json:"name"`
}

// func (Folders) TableName() string {
// 	return "folders" // Nama tabel di database
// }
