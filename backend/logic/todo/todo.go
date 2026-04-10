package todo

import (
	"voice-assistant/backend/domain/todo/todo"
	"voice-assistant/backend/logic"

	"github.com/gin-gonic/gin"
)

type TodoLogic struct {
	logic.BaseLogic
}

func NewTodoLogic(ctx *gin.Context) *TodoLogic {
	return &TodoLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// Create 创建待办
func (l *TodoLogic) Create(req *todo.CreateTodo) (*todo.TodoEntity, error) {
	entity := todo.NewTodoEntity(l.Ctx)
	_, err := entity.SetData(req)
	if err != nil {
		return nil, err
	}

	if err := entity.Validate(); err != nil {
		return nil, err
	}

	res, err := entity.Create()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Update 更新待办
func (l *TodoLogic) Update(req *todo.UpdateTodo) (*todo.TodoEntity, error) {
	entity := todo.NewTodoEntity(l.Ctx)
	_, err := entity.LoadById(uint64(req.ID))
	if err != nil {
		return nil, err
	}

	_, err = entity.SetData(req)
	if err != nil {
		return nil, err
	}

	res, err := entity.Update()
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Get 获取单个待办
func (l *TodoLogic) Get(id uint) (*todo.TodoEntity, error) {
	entity := todo.NewTodoEntity(l.Ctx)
	res, err := entity.LoadById(uint64(id))
	if err != nil {
		return nil, err
	}
	return res, nil
}

// List 获取待办列表
func (l *TodoLogic) List(req *todo.SearchTodo) (*todo.ListResp, error) {
	entity := todo.NewTodoEntity(l.Ctx)

	// 构建查询条件
	cond := entity.MakeConditon(*req)

	total, err := entity.Count(cond)
	if err != nil {
		return nil, err
	}

	list, err := entity.List(cond)
	if err != nil {
		return nil, err
	}

	return &todo.ListResp{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		List:     list,
	}, nil
}

// Del 删除待办
func (l *TodoLogic) Del(req *todo.DelTodo) error {
	entity := todo.NewTodoEntity(l.Ctx)
	// 转换 []uint 为 []uint64
	ids := make([]uint64, len(req.IDs))
	for i, id := range req.IDs {
		ids[i] = uint64(id)
	}
	err := entity.Del(ids...)
	return err
}
