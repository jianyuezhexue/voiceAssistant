package knowledge

import (
	"github.com/gin-gonic/gin"
	"voice-assistant/backend/domain/knowledge/knowledge"
	"voice-assistant/backend/logic"
)

// KnowledgeLogic 知识点逻辑层
type KnowledgeLogic struct {
	logic.BaseLogic
}

// NewKnowledgeLogic 创建知识点逻辑层实例
func NewKnowledgeLogic(ctx *gin.Context) *KnowledgeLogic {
	return &KnowledgeLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// Create 创建知识点
func (l *KnowledgeLogic) Create(req *knowledge.CreateKnowledge) (*knowledge.KnowledgeEntity, error) {
	entity := knowledge.NewKnowledgeEntity(l.Ctx)
	_, err := entity.SetData(req)
	if err != nil {
		return nil, err
	}

	if err := entity.Validate(); err != nil {
		return nil, err
	}

	// TODO: 生成向量嵌入并存储到 Milvus

	res, err := entity.Create()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Update 更新知识点
func (l *KnowledgeLogic) Update(req *knowledge.UpdateKnowledge) (*knowledge.KnowledgeEntity, error) {
	entity := knowledge.NewKnowledgeEntity(l.Ctx)
	_, err := entity.LoadById(uint64(req.ID))
	if err != nil {
		return nil, err
	}

	_, err = entity.SetData(req)
	if err != nil {
		return nil, err
	}

	// TODO: 更新向量嵌入

	res, err := entity.Update()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Get 获取单个知识点
func (l *KnowledgeLogic) Get(id uint) (*knowledge.KnowledgeEntity, error) {
	entity := knowledge.NewKnowledgeEntity(l.Ctx)
	res, err := entity.LoadById(uint64(id))
	if err != nil {
		return nil, err
	}
	return res, nil
}

// List 获取知识点列表
func (l *KnowledgeLogic) List(req *knowledge.SearchKnowledge) (*knowledge.ListResp, error) {
	entity := knowledge.NewKnowledgeEntity(l.Ctx)

	cond := entity.MakeConditon(*req)

	total, err := entity.Count(cond)
	if err != nil {
		return nil, err
	}

	list, err := entity.List(cond)
	if err != nil {
		return nil, err
	}

	return &knowledge.ListResp{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		List:     list,
	}, nil
}

// VectorSearch 向量检索知识点
func (l *KnowledgeLogic) VectorSearch(req *knowledge.VectorSearch) ([]*knowledge.KnowledgeEntity, error) {
	// TODO: 调用 Milvus 进行向量检索
	// 1. 将查询文本转为向量
	// 2. 在 Milvus 中搜索相似向量
	// 3. 根据 VectorID 从 MySQL 获取完整知识点信息
	return nil, nil
}

// Del 删除知识点
func (l *KnowledgeLogic) Del(req *knowledge.DelKnowledge) error {
	entity := knowledge.NewKnowledgeEntity(l.Ctx)
	// TODO: 同时从 Milvus 删除向量
	ids := make([]uint64, len(req.IDs))
	for i, id := range req.IDs {
		ids[i] = uint64(id)
	}
	err := entity.Del(ids...)
	return err
}