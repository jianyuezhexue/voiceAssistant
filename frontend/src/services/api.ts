import type { Todo, Knowledge, ApiResponse } from '../types';

const API_BASE = ''; // Vite dev server proxy handles CORS

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

  if (!response.ok) {
    throw new Error(`API Error: ${response.status}`);
  }

  const result: ApiResponse<T> = await response.json();
  return result.data;
}

export const todoApi = {
  list: () => fetchApi<Todo[]>('/api/v1/todos/list'),

  create: (data: Partial<Todo>) =>
    fetchApi<Todo>('/api/v1/todos', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: Partial<Todo>) =>
    fetchApi<Todo>(`/api/v1/todos/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    fetchApi<void>(`/api/v1/todos?id=${id}`, {
      method: 'DELETE',
    }),
};

export const knowledgeApi = {
  list: () => fetchApi<Knowledge[]>('/api/v1/knowledge/list'),

  create: (data: Partial<Knowledge>) =>
    fetchApi<Knowledge>('/api/v1/knowledge', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    fetchApi<void>(`/api/v1/knowledge?id=${id}`, {
      method: 'DELETE',
    }),
};
