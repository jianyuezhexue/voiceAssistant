<script setup lang="ts">
import { ref } from 'vue';
import HomePage from './pages/HomePage.vue';
import TodoPage from './pages/TodoPage.vue';
import KnowledgePage from './pages/KnowledgePage.vue';

type PageName = 'home' | 'todo' | 'knowledge';

const currentPage = ref<PageName>('home');

function switchPage(page: PageName) {
  currentPage.value = page;
}
</script>

<template>
  <div class="app-container">
    <!-- Warm Cream Header -->
    <header class="app-header">
      <div class="header-left">
        <h1 class="app-title">VoiceAssistant</h1>
      </div>

      <div class="header-right">
        <nav class="header-nav">
          <button
            class="nav-text"
            :class="{ active: currentPage === 'home' }"
            @click="switchPage('home')"
          >
            首页
          </button>
          <button
            class="nav-text"
            :class="{ active: currentPage === 'todo' }"
            @click="switchPage('todo')"
          >
            待办
          </button>
          <button
            class="nav-text"
            :class="{ active: currentPage === 'knowledge' }"
            @click="switchPage('knowledge')"
          >
            知识库
          </button>
        </nav>
        <div class="avatar" title="个人中心"></div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="app-main">
      <transition name="page" mode="out-in">
        <HomePage v-if="currentPage === 'home'" key="home" />
        <TodoPage v-else-if="currentPage === 'todo'" key="todo" />
        <KnowledgePage v-else-if="currentPage === 'knowledge'" key="knowledge" />
      </transition>
    </main>
  </div>
</template>

<style scoped>
.app-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.app-header {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  height: 72px;
  background: rgba(254, 247, 237, 0.95);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--color-border);
}

.header-left {
  flex: 1;
}

.app-title {
  font-family: var(--font-sans);
  font-size: 20px;
  font-weight: 700;
  margin: 0;
  background: linear-gradient(135deg, var(--color-accent) 0%, var(--color-accent-light) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: -0.5px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 28px;
}

.header-nav {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px;
  background: rgba(249, 115, 22, 0.06);
  border-radius: 12px;
}

.nav-text {
  font-family: var(--font-sans);
  font-weight: 500;
  font-size: 14px;
  color: var(--color-text-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 10px 16px;
  border-radius: 8px;
  position: relative;
}

.nav-text:hover {
  color: var(--color-accent);
  background: rgba(249, 115, 22, 0.08);
}

.nav-text.active {
  color: white;
  background: linear-gradient(135deg, var(--color-accent) 0%, var(--color-accent-light) 100%);
  box-shadow: 0 2px 8px rgba(249, 115, 22, 0.25);
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-accent) 0%, var(--color-accent-light) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 16px;
  box-shadow: 0 4px 14px rgba(249, 115, 22, 0.25);
  cursor: pointer;
  transition: all 0.2s ease;
  border: 2px solid transparent;
}

.avatar:hover {
  transform: scale(1.05);
  box-shadow: 0 6px 20px rgba(249, 115, 22, 0.35);
  border-color: var(--color-accent-glow);
}

.app-main {
  flex: 1;
  padding: 28px 32px;
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}

/* Page transitions */
.page-enter-active,
.page-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

@media (max-width: 768px) {
  .app-header {
    padding: 0 16px;
    height: 64px;
  }

  .header-right {
    gap: 16px;
  }

  .header-nav {
    gap: 4px;
    padding: 4px;
  }

  .nav-text {
    font-size: 13px;
    padding: 8px 12px;
  }

  .app-main {
    padding: 16px;
  }

  .avatar {
    width: 36px;
    height: 36px;
  }
}
</style>
