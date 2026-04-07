package todo

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"voice-assistant/backend/api"
	"voice-assistant/backend/domain/todo/todo"
	todoLogic "voice-assistant/backend/logic/todo"
)

// Todo API 控制器
type Todo struct {
	api.Base
}

// NewTodo 创建 Todo API 实例
func NewTodo() *Todo {
	return &Todo{}
}

// Create 创建待办
func (a *Todo) Create(ctx *gin.Context) {
	req := &todo.CreateTodo{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := todoLogic.NewTodoLogic(ctx)
	res, err := logic.Create(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "创建成功")
}

// Update 更新待办
func (a *Todo) Update(ctx *gin.Context) {
	req := &todo.UpdateTodo{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := todoLogic.NewTodoLogic(ctx)
	res, err := logic.Update(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "更新成功")
}

// Get 获取单个待办
func (a *Todo) Get(ctx *gin.Context) {
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
	logic := todoLogic.NewTodoLogic(ctx)
	res, err := logic.Get(uint(id))
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "查询成功")
}

// List 待办列表
func (a *Todo) List(ctx *gin.Context) {
	req := &todo.SearchTodo{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := todoLogic.NewTodoLogic(ctx)
	res, err := logic.List(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(res, "查询成功")
}

// Del 删除待办
func (a *Todo) Del(ctx *gin.Context) {
	req := &todo.DelTodo{}
	if err := a.Bind(ctx, req); err != nil {
		a.Error(err)
		return
	}
	logic := todoLogic.NewTodoLogic(ctx)
	err := logic.Del(req)
	if err != nil {
		a.Error(err)
		return
	}
	a.Success(nil, "删除成功")
}
