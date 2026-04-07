package knowledge

// CreateKnowledge 创建知识点 DTO
type CreateKnowledge struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Category string `json:"category"`
	Source   string `json:"source"`
}

// UpdateKnowledge 更新知识点 DTO
type UpdateKnowledge struct {
	ID       uint   `json:"id" binding:"required"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
}

// SearchKnowledge 搜索知识点 DTO
type SearchKnowledge struct {
	ID        uint     `json:"id" search:"type:eq;column:id;table:knowledge"`
	Page      int64    `json:"page" search:"page"`
	PageSize  int64    `json:"page_size" search:"pageSize"`
	Category  string   `json:"category" search:"type:eq;column:category;table:knowledge"`
	Title     string   `json:"title" search:"type:like;column:title;table:knowledge"`
	Keyword   string   `json:"keyword" search:"type:like;column:content;table:knowledge"`
	CreatedAt []string `json:"created_at" search:"type:between;column:created_at;table:knowledge"`
}

// DelKnowledge 删除知识点 DTO
type DelKnowledge struct {
	IDs []uint `json:"ids" binding:"required"`
}

// VectorSearch 向量检索 DTO
type VectorSearch struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

// ListResp 列表返回
type ListResp struct {
	Page     int64              `json:"page"`
	PageSize int64              `json:"page_size"`
	Total    int64              `json:"total"`
	List     []*KnowledgeEntity `json:"list"`
}
