export interface Todo {
  id: number;
  title: string;
  completed: boolean;
  status: 'pending' | 'in_progress' | 'completed';
  created_at: string;
  updated_at: string;
}

export interface Knowledge {
  id: number;
  title: string;
  content: string;
  summary?: string;
  created_at: string;
  updated_at: string;
}

export interface ApiResponse<T> {
  code: number;
  data: T;
  message: string;
}

export interface ASRMessage {
  text?: string;
  error?: string;
  type?: 'transcript' | 'todo' | 'knowledge';
  data?: Todo | Knowledge;
}
