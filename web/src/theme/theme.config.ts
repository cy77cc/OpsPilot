export const theme = {
  colors: {
    // 背景色
    background: {
      primary: '#0B0E14',    // 一级容器
      secondary: '#161B22'    // 卡片/组件背景
    },
    // 文本色
    text: {
      primary: '#FFFFFF',     // 主要文本
      secondary: '#E5E7EB',   // 次要文本（对比度8.5:1）
      tertiary: '#D1D5DB',    // 三级文本（对比度7.2:1）
      quaternary: '#9CA3AF'   // 四级文本（对比度4.5:1）
    },
    // 品牌色
    brand: {
      primary: '#1677FF'      // DevOps Blue
    },
    // 状态色
    status: {
      running: '#52C41A',     // 运行中
      warning: '#FAAD14',     // 警告
      error: '#FF4D4F',       // 故障
      offline: '#8C8C8C'      // 离线
    }
  },
  // 状态映射
  statusMap: {
    RUNNING: 'running',
    WARNING: 'warning',
    ERROR: 'error',
    OFFLINE: 'offline'
  },
  // 动画配置
  animations: {
    pulse: {
      duration: 2000,         // 动画周期 2 秒
      fadeIn: 500,            // 淡入 0.5 秒
      fadeOut: 1500           // 淡出 1.5 秒
    }
  }
};