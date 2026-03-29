package todo

// CreateTodo 创建待办 DTO
type CreateTodo struct {
	Title     string `json:"title" binding:"required"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	MeetingID *uint  `json:"meeting_id"`
}

// UpdateTodo 更新待办 DTO
type UpdateTodo struct {
	ID      uint   `json:"id" binding:"required"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

// SearchTodo 搜索待办 DTO
type SearchTodo struct {
	ID        uint   `json:"id" search:"type:eq;column:id;table:todos"`
	Page      int64  `json:"page" search:"page"`
	PageSize  int64  `json:"page_size" search:"pageSize"`
	Status    string `json:"status" search:"type:eq;column:status;table:todos"`
	Source    string `json:"source" search:"type:eq;column:source;table:todos"`
	Title     string `json:"title" search:"type:like;column:title;table:todos"`
	CreatedAt []string `json:"created_at" search:"type:between;column:created_at;table:todos"`
}

// DelTodo 删除待办 DTO
type DelTodo struct {
	IDs []uint `json:"ids" binding:"required"`
}

// ListResp 列表返回
type ListResp struct {
	Page     int64         `json:"page"`
	PageSize int64         `json:"page_size"`
	Total    int64         `json:"total"`
	List     []*TodoEntity `json:"list"`
}