<script setup lang="ts">
/**
 * VoiceWaveform.vue - 波形动画组件
 * Canvas 波形动画，60fps 平滑动画，状态联动
 */
import { ref, onMounted, onUnmounted, watch, computed } from 'vue';
import { VoiceState } from '../../types';

interface Props {
  state: VoiceState;
  width?: number;
  height?: number;
  barCount?: number;
  color?: string;
  secondaryColor?: string;
}

const props = withDefaults(defineProps<Props>(), {
  width: 200,
  height: 60,
  barCount: 20,
  color: '#f97316', // orange-500
  secondaryColor: '#fed7aa' // orange-200
});

const canvasRef = ref<HTMLCanvasElement | null>(null);
let animationId: number | null = null;
let ctx: CanvasRenderingContext2D | null = null;

// 波形数据
const bars = ref<number[]>([]);

// 目标值
const targetValues = ref<number[]>([]);

// 初始化波形数据
const initBars = () => {
  bars.value = Array(props.barCount).fill(0).map(() => Math.random() * 0.3 + 0.1);
  targetValues.value = Array(props.barCount).fill(0).map(() => Math.random() * 0.3 + 0.1);
};

// 根据状态更新目标值
const updateTargets = () => {
  const baseHeight = 0.15;

  switch (props.state) {
    case VoiceState.LISTENING:
      // 监听状态：中等高度的波形，模拟环境音
      targetValues.value = Array(props.barCount).fill(0).map(() =>
        Math.random() * 0.2 + baseHeight
      );
      break;

    case VoiceState.RECOGNIZING:
      // 识别状态：波形开始起伏
      targetValues.value = Array(props.barCount).fill(0).map(() =>
        Math.random() * 0.4 + 0.2
      );
      break;

    case VoiceState.THINKING:
      // 思考状态：波浪式动画
      const time = Date.now() / 1000;
      targetValues.value = Array(props.barCount).fill(0).map((_, i) => {
        const wave = Math.sin(time * 2 + i * 0.3) * 0.15;
        return Math.max(0.1, 0.3 + wave);
      });
      break;

    case VoiceState.RESPONDING:
    case VoiceState.PLAYING:
      // 回复/播放状态：活跃的波形
      const t = Date.now() / 1000;
      targetValues.value = Array(props.barCount).fill(0).map((_, i) => {
        const wave = Math.sin(t * 4 + i * 0.5) * 0.25;
        return Math.max(0.1, 0.4 + wave);
      });
      break;

    case VoiceState.IDLE:
    case VoiceState.ERROR:
    default:
      // 空闲/错误状态：平静的波形或归零
      targetValues.value = Array(props.barCount).fill(0).map(() =>
        props.state === VoiceState.ERROR ? 0 : baseHeight
      );
      break;
  }
};

// 动画循环
const animate = () => {
  if (!ctx || !canvasRef.value) return;

  const { width, height } = canvasRef.value;
  const barWidth = width / props.barCount;
  const gap = 2;

  // 清空画布
  ctx.clearRect(0, 0, width, height);

  // 更新波形值（平滑过渡）
  for (let i = 0; i < props.barCount; i++) {
    const target = targetValues.value[i] || 0.1;
    const current = bars.value[i] || 0.1;
    bars.value[i] = current + (target - current) * 0.15;
  }

  // 绘制波形
  const centerY = height / 2;

  bars.value.forEach((value, index) => {
    const barHeight = value * height;
    const x = index * barWidth + gap;
    const y = centerY - barHeight / 2;

    // 渐变效果
    const gradient = ctx!.createLinearGradient(x, y, x, y + barHeight);
    gradient.addColorStop(0, props.color);
    gradient.addColorStop(1, props.secondaryColor);

    ctx!.fillStyle = gradient;
    ctx!.beginPath();
    ctx!.roundRect(x, y, barWidth - gap, barHeight, 2);
    ctx!.fill();
  });

  // 继续动画
  animationId = requestAnimationFrame(animate);
};

// 开始动画
const startAnimation = () => {
  if (animationId) return;
  initBars();
  animate();
};

// 停止动画
const stopAnimation = () => {
  if (animationId) {
    cancelAnimationFrame(animationId);
    animationId = null;
  }
};

// 监听状态变化
watch(() => props.state, (newState) => {
  if (newState === VoiceState.IDLE) {
    stopAnimation();
    // 绘制静止状态
    setTimeout(() => {
      if (canvasRef.value) {
        ctx = canvasRef.value.getContext('2d');
        if (ctx) {
          const { width, height } = canvasRef.value;
          ctx.clearRect(0, 0, width, height);
          // 绘制平静波形
          const barWidth = width / props.barCount;
          const centerY = height / 2;
          bars.value.forEach((value, index) => {
            const barHeight = value * height;
            const x = index * barWidth + 2;
            ctx!.fillStyle = props.secondaryColor;
            ctx!.beginPath();
            ctx!.roundRect(x, centerY - barHeight / 2, barWidth - 2, barHeight, 2);
            ctx!.fill();
          });
        }
      }
    }, 50);
  } else {
    startAnimation();
  }
}, { immediate: true });

// 监听宽度变化
watch(() => props.width, () => {
  if (canvasRef.value) {
    canvasRef.value.width = props.width;
  }
});

onMounted(() => {
  if (canvasRef.value) {
    ctx = canvasRef.value.getContext('2d');
    canvasRef.value.width = props.width;
    canvasRef.value.height = props.height;

    if (props.state !== VoiceState.IDLE) {
      startAnimation();
    }
  }
});

onUnmounted(() => {
  stopAnimation();
});
</script>

<template>
  <div class="voice-waveform-container">
    <canvas
      ref="canvasRef"
      :width="width"
      :height="height"
      class="voice-waveform-canvas"
    />
  </div>
</template>

<style scoped>
.voice-waveform-container {
  display: flex;
  align-items: center;
  justify-content: center;
}

.voice-waveform-canvas {
  display: block;
}
</style>
