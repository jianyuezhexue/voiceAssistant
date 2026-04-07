package meeting

import (
	"github.com/gin-gonic/gin"
	"github.com/jianyuezhexue/base"

	"voice-assistant/backend/component/db"
)

// MeetingInterface 业务模型接口定义
type MeetingInterface interface {
	base.BaseModelInterface[MeetingEntity]
}

// MeetingEntity 会议记录实体
type MeetingEntity struct {
	base.BaseModel[MeetingEntity]
	Title      string  `gorm:"size:255" json:"title"`
	StartTime  *string `json:"start_time"`
	EndTime    *string `json:"end_time"`
	Transcript string  `gorm:"type:text" json:"transcript"` // 完整转录文本
}

// TableName 数据表名
func (m *MeetingEntity) TableName() string {
	return "meetings"
}

// NewMeetingEntity 创建会议实体实例
func NewMeetingEntity(ctx *gin.Context, opt ...base.Option[MeetingEntity]) MeetingInterface {
	entity := &MeetingEntity{}
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
func (m *MeetingEntity) Validate() error {
	// TODO: 实现具体校验逻辑
	return nil
}

// Repair 数据修复
func (m *MeetingEntity) Repair() error {
	// TODO: 实现数据修复逻辑
	return nil
}

// Complete 数据完善
func (m *MeetingEntity) Complete() error {
	// TODO: 实现数据完善逻辑
	return nil
}
