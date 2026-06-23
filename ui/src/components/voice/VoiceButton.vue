<script setup lang="ts">
/**
 * VoiceButton.vue - 语音按钮组件
 * 64px 圆形按钮，渐变橙色背景，状态样式联动
 */
import { computed } from 'vue';
import { VoiceState } from '../../types';

interface Props {
  state: VoiceState;
  disabled?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  size: 'md'
});

const emit = defineEmits<{
  click: [];
}>();

// 尺寸映射
const sizeMap = {
  sm: 'w-12 h-12',
  md: 'w-16 h-16',
  lg: 'w-20 h-20'
};

// 计算按钮尺寸
const buttonSize = computed(() => sizeMap[props.size]);

// 计算状态样式
const stateClasses = computed(() => {
  const baseClasses = 'rounded-full flex items-center justify-center transition-all duration-300 ease-out';

  if (props.disabled) {
    return `${baseClasses} bg-gray-300 cursor-not-allowed`;
  }

  switch (props.state) {
    case VoiceState.IDLE:
    case VoiceState.LISTENING:
      return `${baseClasses} bg-gradient-to-br from-orange-400 to-orange-600 hover:from-orange-500 hover:to-orange-700 shadow-lg hover:shadow-xl cursor-pointer`;
    case VoiceState.RECOGNIZING:
    case VoiceState.THINKING:
      return `${baseClasses} bg-gradient-to-br from-orange-500 to-orange-700 shadow-lg animate-pulse cursor-pointer`;
    case VoiceState.RESPONDING:
    case VoiceState.PLAYING:
      return `${baseClasses} bg-gradient-to-br from-orange-600 to-orange-800 shadow-lg animate-pulse cursor-pointer`;
    case VoiceState.ERROR:
      return `${baseClasses} bg-gradient-to-br from-red-400 to-red-600 shadow-lg cursor-pointer`;
    default:
      return `${baseClasses} bg-gradient-to-br from-orange-400 to-orange-600 shadow-lg cursor-pointer`;
  }
});

// 计算图标
const iconClass = computed(() => {
  switch (props.state) {
    case VoiceState.IDLE:
      return 'w-6 h-6 text-white';
    case VoiceState.LISTENING:
      return 'w-7 h-7 text-white';
    case VoiceState.RECOGNIZING:
    case VoiceState.THINKING:
      return 'w-6 h-6 text-white animate-spin';
    case VoiceState.RESPONDING:
    case VoiceState.PLAYING:
      return 'w-6 h-6 text-white';
    case VoiceState.ERROR:
      return 'w-6 h-6 text-white';
    default:
      return 'w-6 h-6 text-white';
  }
});

// 图标 SVG 路径
const microphonePath = 'M12 14c1.66 0 3-1.34 3-3V5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm-1-9c0-.55.45-1 1-1s1 .45 1 1v6c0 .55-.45 1-1 1s-1-.45-1-1V5zm6 6c0 2.76-2.24 5-5 5s-5-2.24-5-5H5c0 3.53 2.61 6.43 6 6.92V21h2v-3.08c3.39-.49 6-3.39 6-6.92h-2z';

const loadingPath = 'M12 4V2A10 10 0 0 0 2 12h2a8 8 0 0 1 8-8z';

const speakerPath = 'M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z';

const errorPath = 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z';

const iconPath = computed(() => {
  switch (props.state) {
    case VoiceState.IDLE:
    case VoiceState.LISTENING:
      return microphonePath;
    case VoiceState.RECOGNIZING:
    case VoiceState.THINKING:
      return loadingPath;
    case VoiceState.RESPONDING:
    case VoiceState.PLAYING:
      return speakerPath;
    case VoiceState.ERROR:
      return errorPath;
    default:
      return microphonePath;
  }
});

const handleClick = () => {
  if (!props.disabled) {
    emit('click');
  }
};
</script>

<template>
  <button
    :class="[buttonSize, stateClasses]"
    :disabled="disabled"
    @click="handleClick"
    aria-label="Voice button"
  >
    <svg :class="iconClass" viewBox="0 0 24 24" fill="currentColor">
      <path :d="iconPath" />
    </svg>
  </button>
</template>

<style scoped>
button:active:not(:disabled) {
  transform: scale(0.95);
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.8;
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
</style>
