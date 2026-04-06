<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { knowledgeApi } from '../services/api';
import type { Knowledge } from '../types';

const knowledgeItems = ref<Knowledge[]>([]);
const isLoading = ref(true);
const searchQuery = ref('');

const filteredItems = computed(() => {
  if (!searchQuery.value.trim()) {
    return knowledgeItems.value;
  }
  const query = searchQuery.value.toLowerCase();
  return knowledgeItems.value.filter(
    item =>
      item.title.toLowerCase().includes(query) ||
      item.content.toLowerCase().includes(query)
  );
});

async function loadKnowledge() {
  try {
    const data = await knowledgeApi.list();
    knowledgeItems.value = data || [];
  } catch (e) {
    console.error('Failed to load knowledge:', e);
  } finally {
    isLoading.value = false;
  }
}

async function deleteKnowledge(id: number) {
  try {
    await knowledgeApi.delete(id);
    knowledgeItems.value = knowledgeItems.value.filter((k) => k.id !== id);
  } catch (e) {
    console.error('Failed to delete knowledge:', e);
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

function highlightMatch(text: string): string {
  if (!searchQuery.value.trim()) return text;
  const regex = new RegExp(`(${searchQuery.value})`, 'gi');
  return text.replace(regex, '<mark>$1</mark>');
}

onMounted(() => {
  loadKnowledge();
});
</script>

<template>
  <div class="knowledge-page">
    <!-- Warm Cream Background -->
    <div class="ambient-bg">
      <div class="cream-gradient"></div>
    </div>

    <div class="page-content">
      <!-- Page Header -->
      <div class="page-header">
        <div class="header-text">
          <h1 class="page-title">知识库</h1>
          <p class="page-subtitle">管理和查看您的知识内容</p>
        </div>
        <div class="header-badge">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
          </svg>
          <span>{{ knowledgeItems.length }} 条知识</span>
        </div>
      </div>

      <!-- Search Bar -->
      <div class="search-card">
        <div class="search-wrapper">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/>
            <path d="M21 21l-4.35-4.35"/>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            class="search-input"
            placeholder="搜索知识内容..."
          />
          <div v-if="searchQuery" class="search-clear" @click="searchQuery = ''">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12"/>
            </svg>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="isLoading" class="loading-state">
        <div class="loading-spinner"></div>
        <p>加载中...</p>
      </div>

      <!-- Empty State -->
      <div v-else-if="knowledgeItems.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
          </svg>
        </div>
        <h2>暂无知识内容</h2>
        <p>开始说话自动提取知识内容</p>
      </div>

      <!-- No Search Results -->
      <div v-else-if="filteredItems.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="11" cy="11" r="8"/>
            <path d="M21 21l-4.35-4.35"/>
          </svg>
        </div>
        <h2>未找到匹配结果</h2>
        <p>尝试其他搜索关键词</p>
      </div>

      <!-- Knowledge Grid -->
      <div v-else class="knowledge-grid">
        <TransitionGroup name="card">
          <div
            v-for="(item, index) in filteredItems"
            :key="item.id"
            class="knowledge-card"
            :style="{ '--delay': `${index * 0.04}s` }"
          >
            <div class="card-header">
              <div class="card-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                </svg>
              </div>
              <button class="card-delete" @click="deleteKnowledge(item.id)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
            <h3 class="card-title" v-html="highlightMatch(item.title)"></h3>
            <p class="card-content" v-html="highlightMatch(item.content)"></p>
            <div class="card-footer">
              <span class="card-date">{{ formatDate(item.created_at) }}</span>
            </div>
          </div>
        </TransitionGroup>
      </div>
    </div>
  </div>
</template>

<style scoped>
.knowledge-page {
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
    radial-gradient(ellipse at 30% 20%, rgba(251, 146, 60, 0.08) 0%, transparent 50%),
    radial-gradient(ellipse at 70% 80%, rgba(249, 115, 22, 0.06) 0%, transparent 50%),
    radial-gradient(ellipse at 50% 50%, rgba(254, 215, 170, 0.04) 0%, transparent 70%);
}

/* Page Content */
.page-content {
  position: relative;
  z-index: 1;
  max-width: 1000px;
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
  margin-bottom: 28px;
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

.header-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: 16px;
  border: 1px solid var(--color-border);
  box-shadow: 0 4px 16px rgba(249, 115, 22, 0.06);
  color: var(--color-primary);
  font-size: 14px;
  font-weight: 500;
}

.header-badge svg {
  width: 18px;
  height: 18px;
}

/* Search Card */
.search-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-radius: 20px;
  border: 1px solid var(--color-border);
  padding: 6px;
  margin-bottom: 28px;
  box-shadow:
    0 4px 20px rgba(249, 115, 22, 0.06),
    0 8px 32px rgba(249, 115, 22, 0.04);
  animation: fadeInUp 0.6s ease-out 0.1s backwards;
}

.search-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
}

.search-icon {
  width: 20px;
  height: 20px;
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 15px;
  font-family: var(--font-sans);
  color: var(--color-text);
  outline: none;
}

.search-input::placeholder {
  color: var(--color-text-muted);
}

.search-clear {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.05);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  color: var(--color-text-muted);
}

.search-clear:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.search-clear svg {
  width: 14px;
  height: 14px;
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

/* Knowledge Grid */
.knowledge-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

/* Knowledge Card */
.knowledge-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: 20px;
  border: 1px solid var(--color-border);
  padding: 24px;
  box-shadow: 0 4px 16px rgba(249, 115, 22, 0.04);
  transition: all 0.3s ease;
  animation: cardIn 0.5s ease-out backwards;
  animation-delay: var(--delay);
}

@keyframes cardIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.knowledge-card:hover {
  box-shadow: 0 8px 32px rgba(249, 115, 22, 0.1);
  transform: translateY(-4px);
  border-color: rgba(249, 115, 22, 0.25);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;
}

.card-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(249, 115, 22, 0.1) 0%, rgba(249, 115, 22, 0.05) 100%);
  border: 1px solid rgba(249, 115, 22, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  transition: all 0.3s ease;
}

.knowledge-card:hover .card-icon {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  color: white;
  box-shadow: 0 4px 14px rgba(249, 115, 22, 0.3);
}

.card-icon svg {
  width: 22px;
  height: 22px;
}

.card-delete {
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

.knowledge-card:hover .card-delete {
  opacity: 1;
}

.card-delete:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
}

.card-delete svg {
  width: 18px;
  height: 18px;
}

.card-title {
  font-family: var(--font-sans);
  font-size: 17px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 12px;
  line-height: 1.4;
  letter-spacing: -0.01em;
}

.card-title :deep(mark) {
  background: rgba(249, 115, 22, 0.2);
  color: var(--color-primary-dark);
  padding: 0 2px;
  border-radius: 2px;
}

.card-content {
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.7;
  margin: 0 0 16px;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-content :deep(mark) {
  background: rgba(249, 115, 22, 0.2);
  color: var(--color-primary-dark);
  padding: 0 2px;
  border-radius: 2px;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.04);
}

.card-date {
  font-size: 12px;
  color: var(--color-text-muted);
}

/* Card Transitions */
.card-enter-active {
  transition: all 0.5s cubic-bezier(0.16, 1, 0.3, 1);
}

.card-leave-active {
  transition: all 0.3s ease;
}

.card-enter-from {
  opacity: 0;
  transform: scale(0.95) translateY(20px);
}

.card-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

.card-move {
  transition: transform 0.5s ease;
}

/* Responsive */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-badge {
    align-self: flex-start;
  }

  .page-content {
    padding: 24px 16px 40px;
  }

  .page-title {
    font-size: 24px;
  }

  .knowledge-grid {
    grid-template-columns: 1fr;
  }

  .card-delete {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .loading-spinner,
  .knowledge-card,
  .empty-state,
  .search-card,
  .page-content {
    animation: none !important;
  }

  .knowledge-card,
  .page-content {
    opacity: 1 !important;
    transform: none !important;
  }

  * {
    transition-duration: 0.01ms !important;
  }
}
</style>
