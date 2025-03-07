package repository

import (
	"streaming_video_service/internal/streaming/domain"

	"gorm.io/gorm"
)

// VideoRepo definition get video info
type VideoRepo interface {
	AutoMigrate() error
	Create(video *domain.Video) error
	GetByID(id uint) (*domain.Video, error)
	Update(video *domain.Video) error
	FindByStatus(status string) ([]domain.Video, error)
	SearchVideos(keyword string) ([]domain.Video, error)
	RecommendVideos(limit int) ([]domain.Video, error)
	// 其他 CRUD ...
}

// CDN 整合說明：
// 在正式上線時，我們可將 MinIO 的 Bucket 設為 Public Read，或利用 Presigned URL 產生短時間有效的存取連結。
// 例如：如果我們的 CDN 域名為 cdn.example.com，則播放 URL 可由以下方式產生：
// hlsURL = "https://cdn.example.com/" + BucketName + "/processed/{videoID}/index.m3u8"
// 這樣播放器就能藉由 CDN 快速取得影片分段檔。

// VideoRepo definition video repo
type videoRepo struct {
	db *gorm.DB
}

// NewVideoRepo create VideoRepo
func NewVideoRepo(db *gorm.DB) VideoRepo {
	return &videoRepo{db: db}
}

// AutoMigrate 是 GORM 提供的一个方法，它会自动根据你定义的模型（在这个例子中是 Video 模型）来更新数据库中的表结构。具体来说，AutoMigrate 会检查数据库中是否存在与模型字段相对应的列，并确保数据库的表结构与模型定义匹配。
// 作用：
//  1. 自动创建表：如果 Video 表在数据库中不存在，AutoMigrate 会自动创建它。
//  2. 自动更新表结构：如果 Video 模型的字段发生变化（比如新增、修改或删除字段），AutoMigrate 会尝试根据模型的变化来更新现有的表结构，确保它与模型保持一致。
//  3. 避免数据丢失：AutoMigrate 会尽量避免删除或改变已有的数据，只会对表结构进行必要的更新。
//
// 注意事项：
//   - AutoMigrate 并不会自动删除数据库中的字段或表。如果你从模型中删除某些字段，AutoMigrate 不会自动删除数据库中的这些字段。
//   - 它适用于开发阶段的数据库迁移，但在生产环境中使用时，需要小心，因为它不适合进行复杂的迁移操作（比如数据转换或字段删除）。
func (r *videoRepo) AutoMigrate() error {
	return r.db.AutoMigrate(&domain.Video{})
}

// Create (video)：这行代码调用了 GORM 的 Create 方法，它会尝试将传入的 video 对象插入到数据库中。如果 video 对象的字段与 Video 表中的字段匹配，GORM 会自动将它们对应并插入数据库。
// •	.Error：GORM 的操作返回一个 error 对象，.Error 是用来获取操作过程中可能产生的错误。如果插入操作成功，error 为 nil，如果失败则返回一个相应的错误信息。
// 主要功能：
// •	插入数据：将 video 实例的内容插入到数据库中 Video 表。
// •	错误处理：如果插入过程中发生错误（例如数据库连接问题、字段类型不匹配等），该方法会返回错误
// 使用场景：
// •	你可以在需要将新的视频记录插入数据库时调用此方法，比如用户上传新视频时。
func (r *videoRepo) Create(video *domain.Video) error {
	return r.db.Create(video).Error
}

// GetByID get Video by id
// 在 GORM 中，First 是用来查询数据库中某个表的第一条符合条件的记录的方法。具体来说，r.DB.First(&v, id) 的作用是根据给定的 id 来查找 Video 表中 第一条匹配 id 的记录，并将结果赋值给 v。
func (r *videoRepo) GetByID(id uint) (*domain.Video, error) {
	var v domain.Video
	if err := r.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// Update 可以使用save和updates
// 1.	如果 video 的 ID 字段已经存在（即记录已经存在于数据库中），则会进行更新操作，更新该记录的所有字段。
// 2.	如果 video 的 ID 不存在（即这是一个新的对象），则会进行插入操作（类似于 INSERT）。
// 3.	更新全部字段：Save 方法会将整个 video 对象的字段作为新的值，更新数据库中对应记录的所有字段（即使某些字段没有变化，也会被更新）。
// 相似性：当传入结构体时，Save 和 Updates 都会更新传入的字段，即使某些字段没有变化。在这种情况下，它们的行为和效益是一样的。
//   - 区别：Save 会创建新的记录（如果 ID 不存在），而 Updates 是专门用来更新已有记录的。如果你确定是更新已有记录，Updates 更符合语义。
//
// 如何避免全字段更新：
// 如果你只想更新特定字段（例如，某个字段发生了变化），而不更新其他字段，可以：
//  1. 使用 Update 更新单个字段：
//     •	Update 只会更新指定的单个字段，比较高效。
//     •	例如，如果你只想更新 title 字段： r.DB.Model(&video).Update("title", video.Title)
func (r *videoRepo) Update(video *domain.Video) error {
	return r.db.Save(video).Error
}

// FindByStatus find videos by status
func (r *videoRepo) FindByStatus(status string) ([]domain.Video, error) {
	var videos []domain.Video
	if err := r.db.Where("status = ?", status).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

// SearchVideos 利用 PostgreSQL 的 ILIKE 實作模糊搜尋（標題或描述包含 keyword）
// LIKE：区分大小写的模糊匹配。在使用 LIKE 时，查询会区分字母的大小写。例如，如果你搜索 "hello"，它只能匹配 "hello"，而不会匹配 "HELLO" 或 "Hello"。
// ILIKE：不区分大小写的模糊匹配。使用 ILIKE 时，它会忽略大小写，能匹配 "hello", "HELLO", "Hello" 等不同大小写的情况。
func (r *videoRepo) SearchVideos(keyword string) ([]domain.Video, error) {
	var videos []domain.Video
	like := "%" + keyword + "%"
	if err := r.db.Where("(title ILIKE ? OR description ILIKE ?) AND status = ?", like, like, domain.VideoReady).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

// RecommendVideos 依照 ViewCount 降序排序，返回熱門影片（簡單推薦）
// 在 GORM 中，Order 方法用于对查询结果进行排序。它接收一个表示排序规则的字符串，并将其应用到查询中。排序规则可以是升序 (ASC) 或降序 (DESC)。
// 先按 view_count 降序，再按 created_at 升序排序：r.DB.Order("view_count DESC, created_at ASC").Find(&videos)
func (r *videoRepo) RecommendVideos(limit int) ([]domain.Video, error) {
	var videos []domain.Video
	// 获取播放次数最多的前10个视频
	if err := r.db.Order("view_count DESC").Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}
