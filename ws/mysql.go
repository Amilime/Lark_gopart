package ws

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

var DB *gorm.DB

// 1. 文档表
type Document struct {
	Id         int64     `gorm:"primaryKey;autoIncrement"`
	Title      string    `gorm:"type:varchar(255)"`
	DocType    int       `gorm:"column:doc_type"`
	FileKey    string    `gorm:"column:file_key;type:varchar(255)"`
	Content    []byte    `gorm:"column:content;type:longblob"`
	OwnerId    int64     `gorm:"column:owner_id"`
	Version    int       `gorm:"column:version;default:1"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
}

func (Document) TableName() string { return "document" }

// 2. 版本表
type DocVersion struct {
	Id              int64     `gorm:"primaryKey"`
	DocId           int64     `gorm:"column:doc_id"`
	VersionNum      int       `gorm:"column:version_num"`
	ContentSnapshot []byte    `gorm:"column:content_snapshow;type:longblob"`
	EditorId        int64     `gorm:"column:editor_id"`
	CreateTime      time.Time `gorm:"column:create_time;autoCreateTime"`
}

func (DocVersion) TableName() string { return "doc_version" }

// 初始化
func InitMySQL() {
	dsn := "root:root@tcp(localhost:3306)/lark_db?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// 打印大写加粗的错误，方便你在控制台一眼看到
		fmt.Println("=========================================")
		fmt.Printf("MySQL 连接失败 错误: %v\n", err)
		fmt.Println("=========================================")
		DB = nil // 明确置空
	} else {
		fmt.Println("MySQL 连接成功")
	}
}

// 安全检查函数
func checkDB() error {
	if DB == nil {
		return fmt.Errorf("数据库未连接，无法存储数据")
	}
	return nil
}

// ---------------------------------------------------------
// 业务方法 (增加了 checkDB 防崩溃)
// ---------------------------------------------------------

func CreateStaticDocument(title string, fileUrl string, ownerId int64) (int64, error) {
	if err := checkDB(); err != nil {
		return 0, err
	} // 安全检查

	doc := Document{
		Title:   title,
		DocType: 0,
		FileKey: fileUrl,
		OwnerId: ownerId,
		Version: 1,
	}
	result := DB.Create(&doc)
	return doc.Id, result.Error
}

func AutoSaveToDocument(docIdStr string) {
	if err := checkDB(); err != nil {
		return
	} // 安全检查

	data := mergeYjsHistory(docIdStr)
	if len(data) == 0 {
		return
	}

	err := DB.Model(&Document{}).
		Where("id = ? AND doc_type = 1", docIdStr).
		Updates(map[string]interface{}{
			"content":     data,
			"update_time": time.Now(),
		}).Error

	if err != nil {
		fmt.Printf(">>> ❌ [自动保存] 失败: %v\n", err)
	} else {
		fmt.Printf(">>> 💾 [自动保存] 成功 DocID=%s\n", docIdStr)
	}
}

func CreateVersionSnapshot(docIdStr string, userId int64, versionNum int) error {
	if err := checkDB(); err != nil {
		return err
	} // 安全检查

	data := mergeYjsHistory(docIdStr)
	if len(data) == 0 {
		return fmt.Errorf("文档内容为空，无法保存版本")
	}

	version := DocVersion{
		DocId:           stringToInt64(docIdStr),
		VersionNum:      versionNum,
		ContentSnapshot: data,
		EditorId:        userId,
	}

	return DB.Create(&version).Error
}

// 辅助函数
func mergeYjsHistory(docIdStr string) []byte {
	fragments := GetYjsHistory(docIdStr)
	if len(fragments) == 0 {
		return nil
	}
	var merged []byte
	for _, frag := range fragments {
		merged = append(merged, frag...)
	}
	return merged
}

func stringToInt64(s string) int64 {
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}
