package todo

import (
	"github.com/gin-gonic/gin"
	"github.com/jianyuezhexue/base"

	"voice-assistant/backend/component/db"
)

// TodoInterface 业务模型接口定义
type TodoInterface interface {
	base.BaseModelInterface[TodoEntity]
}

// TodoEntity 待办事项实体
type TodoEntity struct {
	base.BaseModel[TodoEntity]
	Title     string `gorm:"size:255;not null" json:"title"`
	Content   string `gorm:"type:text" json:"content"`
	Status    string `gorm:"size:20;default:pending" json:"status"` // pending, completed
	Source    string `gorm:"size:50" json:"source"`                 // voice, manual
	MeetingID *uint  `json:"meeting_id"`                            // 关联会议
}

// TableName 数据表名
func (m *TodoEntity) TableName() string {
	return "todos"
}

// NewTodoEntity 创建待办实体实例
func NewTodoEntity(ctx *gin.Context, opt ...base.Option[TodoEntity]) TodoInterface {
	entity := &TodoEntity{}
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
func (m *TodoEntity) Validate() error {
	// TODO: 实现具体校验逻辑
	return nil
}

// Repair 数据修复
func (m *TodoEntity) Repair() error {
	// TODO: 实现数据修复逻辑
	return nil
}

// Complete 数据完善
func (m *TodoEntity) Complete() error {
	// TODO: 实现数据完善逻辑
	return nil
}

// SetData 设置数据
func (m *TodoEntity) SetData(data any) (*TodoEntity, error) {
	// 使用 copier 或手动赋值
	switch v := data.(type) {
	case *CreateTodo:
		m.Title = v.Title
		m.Content = v.Content
		m.Source = v.Source
		m.MeetingID = v.MeetingID
		m.Status = "pending"
	case *UpdateTodo:
		if v.Title != "" {
			m.Title = v.Title
		}
		if v.Content != "" {
			m.Content = v.Content
		}
		if v.Status != "" {
			m.Status = v.Status
		}
	}
	return m, nil
}
