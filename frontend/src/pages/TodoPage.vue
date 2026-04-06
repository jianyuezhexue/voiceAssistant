<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { todoApi } from '../services/api';
import type { Todo } from '../types';

const todos = ref<Todo[]>([]);
const isLoading = ref(true);
const newTodoTitle = ref('');
const isAdding = ref(false);

const completedCount = computed(() => todos.value.filter(t => t.completed).length);
const pendingCount = computed(() => todos.value.filter(t => !t.completed).length);

async function loadTodos() {
  try {
    const data = await todoApi.list();
    todos.value = data || [];
  } catch (e) {
    console.error('Failed to load todos:', e);
  } finally {
    isLoading.value = false;
  }
}

async function addTodo() {
  if (!newTodoTitle.value.trim() || isAdding.value) return;

  isAdding.value = true;
  try {
    const created = await todoApi.create({ title: newTodoTitle.value });
    if (created) {
      todos.value.unshift(created);
    }
    newTodoTitle.value = '';
  } catch (e) {
    console.error('Failed to add todo:', e);
  } finally {
    isAdding.value = false;
  }
}

async function toggleTodoComplete(todo: Todo) {
  try {
    const updated = await todoApi.update(todo.id, {
      completed: !todo.completed,
      status: todo.completed ? 'pending' : 'completed',
    });
    const index = todos.value.findIndex((t) => t.id === todo.id);
    if (index >= 0) {
      todos.value[index] = updated;
    }
  } catch (e) {
    console.error('Failed to update todo:', e);
  }
}

async function deleteTodo(id: number) {
  try {
    await todoApi.delete(id);
    todos.value = todos.value.filter((t) => t.id !== id);
  } catch (e) {
    console.error('Failed to delete todo:', e);
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

onMounted(() => {
  loadTodos();
});
</script>

<template>
  <div class="todo-page">
    <!-- Warm Cream Background -->
    <div class="ambient-bg">
      <div class="cream-gradient"></div>
    </div>

    <div class="page-content">
      <!-- Page Header -->
      <div class="page-header">
        <div class="header-text">
          <h1 class="page-title">待办事项</h1>
          <p class="page-subtitle">高效管理您的任务</p>
        </div>
        <div class="header-stats">
          <div class="stat-item completed">
            <span class="stat-value">{{ completedCount }}</span>
            <span class="stat-label">已完成</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat-item pending">
            <span class="stat-value">{{ pendingCount }}</span>
            <span class="stat-label">待处理</span>
          </div>
        </div>
      </div>

      <!-- Add Todo Form -->
      <div class="add-todo-card">
        <div class="add-todo-form">
          <div class="input-wrapper">
            <input
              v-model="newTodoTitle"
              type="text"
              class="todo-input"
              placeholder="添加新任务..."
              @keyup.enter="addTodo"
            />
            <div class="input-border"></div>
          </div>
          <button
            class="add-btn"
            @click="addTodo"
            :disabled="!newTodoTitle.trim() || isAdding"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 5v14m-7-7h14"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="isLoading" class="loading-state">
        <div class="loading-spinner"></div>
        <p>加载中...</p>
      </div>

      <!-- Empty State -->
      <div v-else-if="todos.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
          </svg>
        </div>
        <h2>暂无待办事项</h2>
        <p>开始说话自动提取待办内容</p>
      </div>

      <!-- Todo List -->
      <div v-else class="todo-list">
        <TransitionGroup name="todo">
          <div
            v-for="(todo, index) in todos"
            :key="todo.id"
            class="todo-item"
            :style="{ '--delay': `${index * 0.03}s` }"
          >
            <button
              class="todo-checkbox"
              :class="{ completed: todo.completed }"
              @click="toggleTodoComplete(todo)"
            >
              <svg
                v-if="todo.completed"
                class="check-icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </button>

            <div class="todo-content">
              <span class="todo-title" :class="{ completed: todo.completed }">
                {{ todo.title }}
              </span>
              <div class="todo-meta">
                <span class="todo-status" :class="todo.status">
                  {{ todo.status === 'completed' ? '已完成' : '待处理' }}
                </span>
                <span class="todo-date">{{ formatDate(todo.created_at) }}</span>
              </div>
            </div>

            <button class="todo-delete" @click="deleteTodo(todo.id)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round"
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </TransitionGroup>
      </div>
    </div>
  </div>
</template>

<style scoped>
.todo-page {
  min-height: calc(100vh - 140px);
  position: relative;
  overflow: hidden;
}

/* Warm Cream Background */
.ambient-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
}

.cream-gradient {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at 80% 20%, rgba(251, 146, 60, 0.08) 0%, transparent 50%),
    radial-gradient(ellipse at 20% 80%, rgba(249, 115, 22, 0.06) 0%, transparent 50%),
    radial-gradient(ellipse at 50% 50%, rgba(254, 215, 170, 0.04) 0%, transparent 70%);
}

/* Page Content */
.page-content {
  position: relative;
  z-index: 1;
  max-width: 700px;
  margin: 0 auto;
  padding: 40px 24px 60px;
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Page Header */
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 32px;
  gap: 24px;
}

.header-text {
  flex: 1;
}

.page-title {
  font-family: var(--font-sans);
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 4px;
  letter-spacing: -0.02em;
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

.header-stats {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 20px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: 16px;
  border: 1px solid var(--color-border);
  box-shadow: 0 4px 16px rgba(249, 115, 22, 0.06);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  line-height: 1;
}

.stat-item.completed .stat-value {
  color: var(--color-primary);
}

.stat-item.pending .stat-value {
  color: var(--color-text);
}

.stat-label {
  font-size: 11px;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-divider {
  width: 1px;
  height: 32px;
  background: var(--color-border);
}

/* Add Todo Card */
.add-todo-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-radius: 20px;
  border: 1px solid var(--color-border);
  padding: 20px;
  margin-bottom: 24px;
  box-shadow:
    0 4px 20px rgba(249, 115, 22, 0.06),
    0 8px 32px rgba(249, 115, 22, 0.04);
  animation: fadeInUp 0.6s ease-out 0.1s backwards;
}

.add-todo-form {
  display: flex;
  align-items: center;
  gap: 12px;
}

.input-wrapper {
  flex: 1;
  position: relative;
}

.todo-input {
  width: 100%;
  padding: 14px 18px;
  border: 2px solid var(--color-border);
  border-radius: 14px;
  font-size: 15px;
  font-family: var(--font-sans);
  background: white;
  color: var(--color-text);
  outline: none;
  transition: all 0.25s ease;
}

.todo-input::placeholder {
  color: var(--color-text-muted);
}

.todo-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 4px rgba(249, 115, 22, 0.08);
}

.input-border {
  position: absolute;
  inset: -2px;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  opacity: 0;
  z-index: -1;
  filter: blur(8px);
  transition: opacity 0.25s ease;
}

.todo-input:focus + .input-border {
  opacity: 0.1;
}

.add-btn {
  width: 48px;
  height: 48px;
  border: none;
  border-radius: 14px;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.25s ease;
  box-shadow: 0 4px 14px rgba(249, 115, 22, 0.25);
  flex-shrink: 0;
}

.add-btn:hover:not(:disabled) {
  transform: scale(1.05);
  box-shadow: 0 6px 20px rgba(249, 115, 22, 0.35);
}

.add-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.add-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.add-btn svg {
  width: 22px;
  height: 22px;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  color: var(--color-text-muted);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(249, 115, 22, 0.1);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
  animation: fadeInUp 0.6s ease-out 0.1s backwards;
}

.empty-icon {
  width: 100px;
  height: 100px;
  border-radius: 28px;
  background: linear-gradient(135deg, rgba(249, 115, 22, 0.1) 0%, rgba(249, 115, 22, 0.05) 100%);
  border: 1px solid rgba(249, 115, 22, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  margin-bottom: 24px;
}

.empty-icon svg {
  width: 44px;
  height: 44px;
  opacity: 0.7;
}

.empty-state h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 8px;
}

.empty-state p {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

/* Todo List */
.todo-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Todo Item */
.todo-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: 16px;
  border: 1px solid var(--color-border);
  box-shadow: 0 2px 12px rgba(249, 115, 22, 0.04);
  transition: all 0.25s ease;
  animation: slideIn 0.4s ease-out backwards;
  animation-delay: var(--delay);
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(-20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.todo-item:hover {
  box-shadow: 0 4px 20px rgba(249, 115, 22, 0.08);
  transform: translateY(-2px);
  border-color: rgba(249, 115, 22, 0.2);
}

/* Todo Checkbox */
.todo-checkbox {
  width: 28px;
  height: 28px;
  border-radius: 10px;
  border: 2px solid var(--color-border);
  background: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.25s ease;
  flex-shrink: 0;
  padding: 0;
}

.todo-checkbox:hover {
  border-color: var(--color-primary);
  background: rgba(249, 115, 22, 0.05);
}

.todo-checkbox.completed {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(249, 115, 22, 0.25);
}

.check-icon {
  width: 16px;
  height: 16px;
  color: white;
}

/* Todo Content */
.todo-content {
  flex: 1;
  min-width: 0;
}

.todo-title {
  display: block;
  font-size: 15px;
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.5;
  word-break: break-word;
  transition: all 0.25s ease;
}

.todo-title.completed {
  text-decoration: line-through;
  color: var(--color-text-muted);
}

.todo-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
}

.todo-status {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  background: rgba(249, 115, 22, 0.08);
  color: var(--color-primary);
  transition: all 0.25s ease;
}

.todo-status.completed {
  background: rgba(249, 115, 22, 0.15);
}

.todo-date {
  font-size: 12px;
  color: var(--color-text-muted);
}

/* Todo Delete */
.todo-delete {
  background: transparent;
  border: none;
  padding: 8px;
  cursor: pointer;
  color: var(--color-text-muted);
  border-radius: 10px;
  transition: all 0.2s ease;
  opacity: 0;
  flex-shrink: 0;
}

.todo-item:hover .todo-delete {
  opacity: 1;
}

.todo-delete:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
}

.todo-delete svg {
  width: 18px;
  height: 18px;
}

/* Todo Transitions */
.todo-enter-active {
  transition: all 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.todo-leave-active {
  transition: all 0.3s ease;
}

.todo-enter-from {
  opacity: 0;
  transform: translateY(-20px);
}

.todo-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.todo-move {
  transition: transform 0.4s ease;
}

/* Responsive */
@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-stats {
    justify-content: center;
  }

  .page-content {
    padding: 24px 16px 40px;
  }

  .page-title {
    font-size: 24px;
  }

  .add-todo-form {
    flex-direction: column;
  }

  .add-btn {
    width: 100%;
    height: 48px;
  }

  .todo-delete {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .loading-spinner,
  .todo-item,
  .empty-state,
  .add-todo-card,
  .page-content {
    animation: none !important;
  }

  .todo-item,
  .page-content {
    opacity: 1 !important;
    transform: none !important;
  }

  * {
    transition-duration: 0.01ms !important;
  }
}
</style>
