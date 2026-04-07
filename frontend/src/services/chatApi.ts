// 基础路径 (通过 Vite 代理)
const API_BASE = '/api/v1'

export interface ChatRequest {
  session_id?: string
  message: string
}

export interface ChatResponse {
  session_id: string
  text: string
  created_at: string
}

export const chatApi = {
  // 发送文字对话
  async sendMessage(req: ChatRequest): Promise<ChatResponse> {
    const resp = await fetch(`${API_BASE}/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req)
    })
    if (!resp.ok) throw new Error(`Chat API error: ${resp.status}`)
    return resp.json()
  }
}
