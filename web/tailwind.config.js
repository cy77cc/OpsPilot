/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // 主色调 (Primary Colors) - Indigo 色系
        primary: {
          50: '#eef2ff',
          100: '#e0e7ff',
          200: '#c7d2fe',
          500: '#6366f1',
          600: '#4f46e5',
          700: '#4338ca',
          900: '#312e81',
        },
        // 中性色 (Neutral Colors) - Gray 色系
        gray: {
          50: '#fafbfc',
          100: '#f8f9fa',
          200: '#e9ecef',
          300: '#dee2e6',
          400: '#ced4da',
          500: '#6c757d',
          700: '#495057',
          900: '#212529',
        },
        // 语义色 (Semantic Colors)
        success: '#10b981',
        warning: '#f59e0b',
        error: '#ef4444',
        info: '#3b82f6',
        // 保留旧的颜色以保持向后兼容
        'bg-primary': '#0B0E14',
        'bg-secondary': '#161B22',
        'text-primary': '#FFFFFF',
        'text-secondary': '#E5E7EB',
        'text-tertiary': '#D1D5DB',
        'text-quaternary': '#9CA3AF',
        'brand-primary': '#1677FF',
        'status-running': '#52C41A',
        'status-warning': '#FAAD14',
        'status-error': '#FF4D4F',
        'status-offline': '#8C8C8C'
      },
      spacing: {
        xs: '4px',
        sm: '8px',
        md: '16px',
        lg: '24px',
        xl: '32px',
        '2xl': '48px',
        '3xl': '64px',
      },
      borderRadius: {
        sm: '4px',
        md: '8px',
        lg: '12px',
        xl: '16px',
        full: '9999px',
      },
      boxShadow: {
        sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
        md: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)',
        lg: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)',
        xl: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1)',
        '2xl': '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)',
      },
      fontFamily: {
        sans: [
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif',
        ],
        mono: [
          'SF Mono',
          'Monaco',
          'Cascadia Code',
          'Consolas',
          'monospace',
        ],
      },
      animation: {
        'status-pulse': 'status-pulse 2s ease-in-out infinite'
      },
      keyframes: {
        'status-pulse': {
          '0%, 100%': {
            opacity: '0.4'
          },
          '50%': {
            opacity: '1'
          }
        }
      }
    },
  },
  plugins: [],
}