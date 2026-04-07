package asr

import (
	"github.com/gin-gonic/gin"
	"voice-assistant/backend/domain/knowledge/knowledge"
	"voice-assistant/backend/domain/todo/todo"
	"voice-assistant/backend/logic"
)

// ASRLogic 语音识别逻辑层
type ASRLogic struct {
	logic.BaseLogic
}

// ASRResult ASR 识别结果
type ASRResult struct {
	Text      string                      `json:"text"`
	Todos     []todo.CreateTodo           `json:"todos"`
	Knowledge []knowledge.CreateKnowledge `json:"knowledge"`
}

// NewASRLogic 创建 ASR 逻辑层实例
func NewASRLogic(ctx *gin.Context) *ASRLogic {
	return &ASRLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// Process 处理语音识别结果
func (l *ASRLogic) Process(text string) (*ASRResult, error) {
	// TODO: 调用 LLM 解析文本内容
	// 1. 将文本发送给大模型
	// 2. 大模型分析并分类内容（待办/知识/日程）
	// 3. 返回结构化结果

	return &ASRResult{
		Text: text,
	}, nil
}

// ParseWithLLM 使用大模型解析文本
func (l *ASRLogic) ParseWithLLM(text string) (*ASRResult, error) {
	// TODO: 实现 LLM 调用
	// 1. 构造 prompt
	// 2. 调用 LLM API
	// 3. 解析返回结果
	return nil, nil
}
