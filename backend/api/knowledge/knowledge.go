package knowledge

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"voice-assistant/backend/api"
	"voice-assistant/backend/domain/knowledge/knowledge"
	knowledgeLogic "voice-assistant/backend/logic/knowledge"
)

// Knowledge API 控制器
type Knowledge struct {
	api.Base
}

// NewKnowledge 创建 Knowledge API 实例
func NewKnowledge() *Knowledge {
	return &Knowledge{}
}

// Create 创建知识点
func (a *Knowledge) Create(ctx *gin.Context) {
	req := &knowledge.CreateKnowledge{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := knowledgeLogic.NewKnowledgeLogic(ctx)
	res, err := logic.Create(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "创建成功")
}

// Update 更新知识点
func (a *Knowledge) Update(ctx *gin.Context) {
	req := &knowledge.UpdateKnowledge{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := knowledgeLogic.NewKnowledgeLogic(ctx)
	res, err := logic.Update(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "更新成功")
}

// Get 获取单个知识点
func (a *Knowledge) Get(ctx *gin.Context) {
	req := &struct{}{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		a.Error(err)
		return
	}
	logic := knowledgeLogic.NewKnowledgeLogic(ctx)
	res, err := logic.Get(uint(id))
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "查询成功")
}

// List 知识点列表
func (a *Knowledge) List(ctx *gin.Context) {
	req := &knowledge.SearchKnowledge{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := knowledgeLogic.NewKnowledgeLogic(ctx)
	res, err := logic.List(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "查询成功")
}

// Search 向量检索知识点
func (a *Knowledge) Search(ctx *gin.Context) {
	req := &knowledge.VectorSearch{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	logic := knowledgeLogic.NewKnowledgeLogic(ctx)
	res, err := logic.VectorSearch(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "检索成功")
}

// Del 删除知识点
func (a *Knowledge) Del(ctx *gin.Context) {
	req := &knowledge.DelKnowledge{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := knowledgeLogic.NewKnowledgeLogic(ctx)
	err := logic.Del(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(nil, "删除成功")
}
