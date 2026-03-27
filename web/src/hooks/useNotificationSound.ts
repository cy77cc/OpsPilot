/**
 * 通知声音工具
 * 用于在新通知到达时播放提示音
 */

// 音频上下文缓存
let audioContext: AudioContext | null = null;

/**
 * 获取或创建 AudioContext
 */
const getAudioContext = (): AudioContext => {
  if (!audioContext) {
    audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
  }
  return audioContext;
};

/**
 * 播放简单的通知提示音
 */
export const playNotificationSound = async (type: 'default' | 'success' | 'warning' | 'error' = 'default'): Promise<void> => {
  try {
    // 检查用户是否允许声音提示
    const soundEnabled = localStorage.getItem('notification:sound:enabled') !== 'false';
    if (!soundEnabled) {
      return;
    }

    const ctx = getAudioContext();

    // 如果音频上下文被暂停，需要恢复
    if (ctx.state === 'suspended') {
      await ctx.resume();
    }

    // 创建振荡器和增益节点
    const oscillator = ctx.createOscillator();
    const gainNode = ctx.createGain();

    // 根据类型设置不同的音调
    const frequencies: Record<string, number[]> = {
      default: [800, 600],
      success: [523, 659],  // C5, E5
      warning: [440, 350],  // A4, F4
      error: [300, 200],    // 低沉的警告音
    };

    const freqs = frequencies[type] || frequencies.default;

    oscillator.connect(gainNode);
    gainNode.connect(ctx.destination);

    oscillator.type = 'sine';
    oscillator.frequency.setValueAtTime(freqs[0], ctx.currentTime);
    oscillator.frequency.setValueAtTime(freqs[1], ctx.currentTime + 0.1);

    // 音量包络
    gainNode.gain.setValueAtTime(0, ctx.currentTime);
    gainNode.gain.linearRampToValueAtTime(0.3, ctx.currentTime + 0.05);
    gainNode.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3);

    // 播放
    oscillator.start(ctx.currentTime);
    oscillator.stop(ctx.currentTime + 0.3);
  } catch (error) {
    console.warn('播放通知声音失败:', error);
  }
};

/**
 * 设置是否启用通知声音
 */
export const setNotificationSoundEnabled = (enabled: boolean): void => {
  localStorage.setItem('notification:sound:enabled', String(enabled));
};

/**
 * 获取通知声音是否启用
 */
export const isNotificationSoundEnabled = (): boolean => {
  return localStorage.getItem('notification:sound:enabled') !== 'false';
};

/**
 * 测试通知声音
 */
export const testNotificationSound = async (): Promise<void> => {
  await playNotificationSound('default');
};

export default {
  playNotificationSound,
  setNotificationSoundEnabled,
  isNotificationSoundEnabled,
  testNotificationSound,
};
