<script setup lang="ts">
/**
 * DialogueBubble.vue - 对话气泡组件
 * 用户/AI 气泡区分，流式文字逐字显示
 */
import { ref, computed, watch, onMounted } from 'vue';

interface Props {
  type: 'user' | 'ai';
  content: string;
  timestamp?: Date;
  isStreaming?: boolean;
  avatar?: string;
}

const props = withDefaults(defineProps<Props>(), {
  timestamp: () => new Date(),
  isStreaming: false,
  avatar: ''
});

// 流式显示的文字
const displayContent = ref('');
let streamIndex = 0;
let streamTimer: ReturnType<typeof setTimeout> | null = null;

// 角色配置
const roleConfig = computed(() => {
  if (props.type === 'user') {
    return {
      bubbleClass: 'bg-gradient-to-br from-orange-500 to-orange-600 text-white rounded-2xl rounded-br-md',
      alignment: 'justify-end',
      avatarBg: 'bg-orange-500',
      avatarText: 'U',
      messageAlign: 'text-right'
    };
  }
  return {
    bubbleClass: 'bg-white border border-gray-200 text-gray-800 rounded-2xl rounded-bl-md shadow-sm',
    alignment: 'justify-start',
    avatarBg: 'bg-gray-100',
    avatarText: 'AI',
    messageAlign: 'text-left'
  };
});

// 格式化时间
const formattedTime = computed(() => {
  const t = props.timestamp;
  return `${t.getHours().toString().padStart(2, '0')}:${t.getMinutes().toString().padStart(2, '0')}`;
});

// 流式文字动画
const startStreaming = () => {
  if (streamTimer) {
    clearTimeout(streamTimer);
    streamTimer = null;
  }

  streamIndex = 0;
  displayContent.value = '';

  const streamChar = () => {
    if (streamIndex < props.content.length) {
      displayContent.value += props.content[streamIndex];
      streamIndex++;
      streamTimer = setTimeout(streamChar, 30); // 30ms per character
    }
  };

  streamChar();
};

// 监听内容变化
watch(() => props.content, (newContent) => {
  if (props.isStreaming && newContent) {
    startStreaming();
  } else {
    displayContent.value = newContent;
  }
}, { immediate: true });

// 停止流式显示
watch(() => props.isStreaming, (streaming) => {
  if (!streaming && streamTimer) {
    clearTimeout(streamTimer);
    streamTimer = null;
    displayContent.value = props.content; // 直接显示完整内容
  }
});

onMounted(() => {
  if (props.isStreaming && props.content) {
    startStreaming();
  } else {
    displayContent.value = props.content;
  }
});
</script>

<template>
  <div :class="['flex gap-3 py-3', roleConfig.alignment]">
    <!-- AI 头像 -->
    <div
      v-if="type === 'ai'"
      :class="[
        'w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium',
        roleConfig.avatarBg
      ]"
    >
      {{ roleConfig.avatarText }}
    </div>

    <!-- 消息内容 -->
    <div class="flex flex-col gap-1 max-w-[75%]">
      <!-- 气泡 -->
      <div :class="['px-4 py-2.5', roleConfig.bubbleClass]">
        <!-- 流式光标 -->
        <span class="inline-block">
          {{ displayContent }}
          <span
            v-if="isStreaming"
            class="inline-block w-1.5 h-4 bg-current ml-0.5 animate-pulse"
          />
        </span>
      </div>

      <!-- 时间戳 -->
      <span
        :class="['text-xs text-gray-400', roleConfig.messageAlign]"
      >
        {{ formattedTime }}
      </span>
    </div>

    <!-- 用户头像 -->
    <div
      v-if="type === 'user'"
      :class="[
        'w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium',
        roleConfig.avatarBg
      ]"
    >
      {{ roleConfig.avatarText }}
    </div>
  </div>
</template>

<style scoped>
/* 气泡动画 */
.bubble-enter-active {
  animation: bubbleIn 0.3s ease-out;
}

.bubble-leave-active {
  animation: bubbleOut 0.2s ease-in;
}

@keyframes bubbleIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(10px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

@keyframes bubbleOut {
  from {
    opacity: 1;
    transform: scale(1);
  }
  to {
    opacity: 0;
    transform: scale(0.95);
  }
}

/* 流式光标动画 */
@keyframes blink {
  0%, 50% {
    opacity: 1;
  }
  51%, 100% {
    opacity: 0;
  }
}

.animate-pulse {
  animation: blink 1s infinite;
}
</style>
