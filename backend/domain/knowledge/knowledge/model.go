package knowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/jianyuezhexue/base"

	"voice-assistant/backend/component/db"
)

// KnowledgeInterface 业务模型接口定义
type KnowledgeInterface interface {
	base.BaseModelInterface[KnowledgeEntity]
}

// KnowledgeEntity 知识点实体
type KnowledgeEntity struct {
	base.BaseModel[KnowledgeEntity]
	Title    string `gorm:"size:255;not null" json:"title"`
	Content  string `gorm:"type:text;not null" json:"content"`
	Category string `gorm:"size:100" json:"category"`  // business, technical, etc.
	Source   string `gorm:"size:50" json:"source"`     // voice, manual
	VectorID string `gorm:"size:255" json:"vector_id"` // Milvus 向量 ID
}

// TableName 数据表名
func (m *KnowledgeEntity) TableName() string {
	return "knowledge"
}

// NewKnowledgeEntity 创建知识点实体实例
func NewKnowledgeEntity(ctx *gin.Context, opt ...base.Option[KnowledgeEntity]) KnowledgeInterface {
	entity := &KnowledgeEntity{}
	entity.BaseModel = base.NewBaseModel(ctx, db.GetDB(), entity.TableName(), entity)

	// 自定义配置选项
	if len(opt) > 0 {
		for _, fc := range opt {
			fc(&entity.BaseModel)
		}
	}
	return entity
}

// Validate 数据校验
func (m *KnowledgeEntity) Validate() error {
	// TODO: 实现具体校验逻辑
	return nil
}

// Repair 数据修复
func (m *KnowledgeEntity) Repair() error {
	// TODO: 实现数据修复逻辑
	return nil
}

// Complete 数据完善
func (m *KnowledgeEntity) Complete() error {
	// TODO: 实现数据完善逻辑
	return nil
}
