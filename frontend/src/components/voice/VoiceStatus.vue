<script setup lang="ts">
/**
 * VoiceStatus.vue - 语音状态显示组件
 * 5种状态文字显示，状态切换动画
 */
import { computed, ref, watch } from 'vue';
import { VoiceState } from '../../types';

interface Props {
  state: VoiceState;
  recognizedText?: string;
  responseText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  recognizedText: '',
  responseText: ''
});

const emit = defineEmits<{
  interrupt: [];
}>();

// 动画状态
const isAnimating = ref(false);
const displayText = ref('');

// 状态映射
const stateConfig = {
  [VoiceState.IDLE]: {
    text: '点击开始说话',
    subtext: '',
    icon: 'mic',
    color: 'text-orange-500'
  },
  [VoiceState.LISTENING]: {
    text: '我在听...',
    subtext: '请说话',
    icon: 'mic-active',
    color: 'text-orange-500'
  },
  [VoiceState.RECOGNIZING]: {
    text: '识别中...',
    subtext: props.recognizedText || '正在识别您说的话',
    icon: 'loading',
    color: 'text-orange-600'
  },
  [VoiceState.THINKING]: {
    text: '思考中...',
    subtext: props.recognizedText || '',
    icon: 'thinking',
    color: 'text-orange-600'
  },
  [VoiceState.RESPONDING]: {
    text: '回复中...',
    subtext: props.responseText || '正在生成回复',
    icon: 'speaker',
    color: 'text-orange-500'
  },
  [VoiceState.PLAYING]: {
    text: '播放中',
    subtext: props.responseText || '',
    icon: 'speaker',
    color: 'text-orange-500'
  },
  [VoiceState.ERROR]: {
    text: '出错了',
    subtext: '请点击重试',
    icon: 'error',
    color: 'text-red-500'
  }
};

// 计算当前状态配置
const currentConfig = computed(() => stateConfig[props.state] || stateConfig[VoiceState.IDLE]);

// 文字动画
watch(() => props.state, (newState, oldState) => {
  if (newState !== oldState) {
    isAnimating.value = true;
    setTimeout(() => {
      isAnimating.value = false;
    }, 300);
  }
});

watch(() => props.responseText, (newText) => {
  if (newText) {
    displayText.value = newText;
  }
}, { immediate: true });

// 图标 SVG 路径
const iconPaths = {
  mic: 'M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm-1-9c0-.55.45-1 1-1s1 .45 1 1v6c0 .55-.45 1-1 1s-1-.45-1-1V5zm6 6c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z',
  'mic-active': 'M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm-1-9c0-.55.45-1 1-1s1 .45 1 1v6c0 .55-.45 1-1 1s-1-.45-1-1V5zm6 6c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z',
  loading: 'M12 4V2A10 10 0 0 0 2 12h2a8 8 0 0 1 8-8z',
  thinking: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z',
  speaker: 'M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z',
  error: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z'
};

const currentIcon = computed(() => iconPaths[currentConfig.value.icon as keyof typeof iconPaths] || iconPaths.mic);
</script>

<template>
  <div class="voice-status-container flex flex-col items-center gap-2">
    <!-- 主状态文字 -->
    <div class="flex items-center gap-3">
      <!-- 图标 -->
      <svg
        :class="[
          'w-5 h-5 transition-colors duration-300',
          currentConfig.color,
          state === VoiceState.RECOGNIZING || state === VoiceState.THINKING ? 'animate-spin' : '',
          state === VoiceState.PLAYING || state === VoiceState.RESPONDING ? 'animate-pulse' : ''
        ]"
        viewBox="0 0 24 24"
        fill="currentColor"
      >
        <path :d="currentIcon" />
      </svg>

      <!-- 文字 -->
      <span
        :class="[
          'text-lg font-medium transition-all duration-300',
          currentConfig.color,
          isAnimating ? 'opacity-0 transform translate-y-1' : 'opacity-100 transform translate-y-0'
        ]"
      >
        {{ currentConfig.text }}
      </span>
    </div>

    <!-- 子文字（识别内容/回复内容） -->
    <div
      v-if="currentConfig.subtext"
      class="text-sm text-gray-500 text-center max-w-xs truncate"
    >
      {{ currentConfig.subtext }}
    </div>

    <!-- 流式回复显示 -->
    <div
      v-if="(state === VoiceState.RESPONDING || state === VoiceState.PLAYING) && responseText"
      class="text-sm text-gray-600 text-center max-w-md px-4 leading-relaxed"
    >
      {{ responseText }}...
    </div>

    <!-- 打断提示 -->
    <button
      v-if="state !== VoiceState.IDLE && state !== VoiceState.ERROR"
      class="mt-2 px-3 py-1 text-xs text-gray-500 hover:text-orange-500 transition-colors"
      @click="emit('interrupt')"
    >
      点击打断
    </button>
  </div>
</template>

<style scoped>
.voice-status-container {
  min-height: 80px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.fade-in {
  animation: fadeIn 0.3s ease-out;
}
</style>
