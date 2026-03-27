/**
 * Ant Design 主题配置
 *
 * 基于设计系统 Token 配置 Ant Design 主题
 */

import type { ThemeConfig } from 'antd';

export const antdTheme: ThemeConfig = {
  token: {
    // ========================================================================
    // 色彩配置
    // ========================================================================
    colorPrimary: '#6366f1',      // 主色 (Indigo 500)
    colorSuccess: '#10b981',      // 成功色
    colorWarning: '#f59e0b',      // 警告色
    colorError: '#ef4444',        // 错误色
    colorInfo: '#3b82f6',         // 信息色

    // 文本颜色
    colorText: '#495057',         // 主要文本 (Gray 700)
    colorTextSecondary: '#6c757d', // 次要文本 (Gray 500)
    colorTextTertiary: '#6c757d', // 三级文本 (Gray 500)
    colorTextQuaternary: '#ced4da', // 四级文本 (Gray 400)

    // 边框颜色
    colorBorder: '#dee2e6',       // 边框 (Gray 300)
    colorBorderSecondary: '#e9ecef', // 次要边框 (Gray 200)

    // 背景颜色
    colorBgContainer: '#ffffff',  // 容器背景
    colorBgElevated: '#ffffff',   // 浮层背景
    colorBgLayout: '#fafbfc',     // 布局背景 (Gray 50)

    // ========================================================================
    // 尺寸配置
    // ========================================================================
    borderRadius: 8,              // 默认圆角 (md)
    borderRadiusLG: 12,           // 大圆角 (lg)
    borderRadiusSM: 4,            // 小圆角 (sm)
    borderRadiusXS: 4,            // 超小圆角

    // ========================================================================
    // 字体配置
    // ========================================================================
    fontSize: 14,                 // 默认字体大小
    fontSizeHeading1: 24,         // H1 字体大小
    fontSizeHeading2: 20,         // H2 字体大小
    fontSizeHeading3: 18,         // H3 字体大小
    fontSizeHeading4: 16,         // H4 字体大小
    fontSizeHeading5: 14,         // H5 字体大小

    fontFamily: [
      '-apple-system',
      'BlinkMacSystemFont',
      'Segoe UI',
      'PingFang SC',
      'Hiragino Sans GB',
      'Microsoft YaHei',
      'sans-serif',
    ].join(', '),

    fontFamilyCode: [
      'SF Mono',
      'Monaco',
      'Cascadia Code',
      'Consolas',
      'monospace',
    ].join(', '),

    // ========================================================================
    // 行高配置
    // ========================================================================
    lineHeight: 1.5,              // 默认行高
    lineHeightHeading1: 1.25,     // H1 行高
    lineHeightHeading2: 1.25,     // H2 行高
    lineHeightHeading3: 1.25,     // H3 行高

    // ========================================================================
    // 间距配置
    // ========================================================================
    padding: 16,                  // 默认内边距 (md)
    paddingLG: 24,                // 大内边距 (lg)
    paddingXL: 32,                // 超大内边距 (xl)
    paddingSM: 12,                // 小内边距
    paddingXS: 8,                 // 超小内边距 (sm)

    margin: 16,                   // 默认外边距
    marginLG: 24,                 // 大外边距
    marginXL: 32,                 // 超大外边距
    marginSM: 12,                 // 小外边距
    marginXS: 8,                  // 超小外边距

    // ========================================================================
    // 边框配置
    // ========================================================================
    lineWidth: 1,                 // 边框宽度
    lineType: 'solid',            // 边框类型

    // ========================================================================
    // 阴影配置
    // ========================================================================
    boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.08)', // 默认阴影 (md)
    boxShadowSecondary: '0 4px 6px -1px rgba(0, 0, 0, 0.1)', // 次要阴影 (lg)

    // ========================================================================
    // 动画配置
    // ========================================================================
    motionDurationFast: '0.15s',  // 快速动画
    motionDurationMid: '0.25s',   // 标准动画
    motionDurationSlow: '0.35s',  // 慢速动画

    motionEaseInOut: 'cubic-bezier(0.4, 0.0, 0.2, 1)',      // 标准缓动
    motionEaseOut: 'cubic-bezier(0.0, 0.0, 0.2, 1)',        // 减速缓动
    motionEaseOutBack: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)', // 弹性缓动

    // ========================================================================
    // 控件高度
    // ========================================================================
    controlHeight: 40,            // 默认控件高度
    controlHeightLG: 48,          // 大控件高度
    controlHeightSM: 32,          // 小控件高度
  },

  components: {
    // ========================================================================
    // Button 组件
    // ========================================================================
    Button: {
      controlHeight: 40,          // 按钮高度
      controlHeightLG: 48,        // 大按钮高度
      controlHeightSM: 32,        // 小按钮高度
      borderRadius: 8,            // 圆角
      fontWeight: 500,            // 字重 (medium)
      primaryShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.05)', // 主按钮阴影
    },

    // ========================================================================
    // Input 组件
    // ========================================================================
    Input: {
      controlHeight: 40,          // 输入框高度
      controlHeightLG: 48,        // 大输入框高度
      controlHeightSM: 32,        // 小输入框高度
      borderRadius: 8,            // 圆角
      paddingBlock: 8,            // 垂直内边距
      paddingInline: 12,          // 水平内边距
    },

    // ========================================================================
    // Card 组件
    // ========================================================================
    Card: {
      borderRadius: 12,           // 圆角 (lg)
      boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.08)', // 阴影 (md)
      paddingLG: 24,              // 内边距
    },

    // ========================================================================
    // Table 组件
    // ========================================================================
    Table: {
      rowHoverBg: '#f8f9fa',      // 行悬停背景 (Gray 100)
      headerBg: '#f8f9fa',        // 表头背景 (Gray 100)
      headerColor: '#495057',     // 表头文字颜色 (Gray 700)
      cellPaddingBlock: 16,       // 单元格垂直内边距
      cellPaddingInline: 16,      // 单元格水平内边距
      borderColor: '#e9ecef',     // 边框颜色 (Gray 200)
      headerSplitColor: '#e9ecef', // 表头分隔线颜色
      rowSelectedBg: '#eef2ff',   // 选中行背景 (Primary 50)
      rowSelectedHoverBg: '#e0e7ff', // 选中行悬停背景 (Primary 100)
    },

    // ========================================================================
    // Modal 组件
    // ========================================================================
    Modal: {
      borderRadius: 12,           // 圆角 (lg)
      boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)', // 阴影 (2xl)
      headerBg: '#ffffff',        // 头部背景
      contentBg: '#ffffff',       // 内容背景
      footerBg: '#ffffff',        // 底部背景
    },

    // ========================================================================
    // Form 组件
    // ========================================================================
    Form: {
      labelFontSize: 14,          // Label 字体大小
      labelColor: '#495057',      // Label 颜色 (Gray 700)
      labelHeight: 40,            // Label 高度
      verticalLabelPadding: '0 0 8px', // 垂直 Label 内边距
      itemMarginBottom: 24,       // Form Item 底部间距
    },

    // ========================================================================
    // Tag 组件
    // ========================================================================
    Tag: {
      borderRadiusSM: 4,          // 圆角 (sm)
      defaultBg: '#f8f9fa',       // 默认背景 (Gray 100)
      defaultColor: '#495057',    // 默认文字颜色 (Gray 700)
    },

    // ========================================================================
    // Notification 组件
    // ========================================================================
    Notification: {
      width: 384,                 // 宽度
      borderRadius: 8,            // 圆角 (md)
    },

    // ========================================================================
    // Message 组件
    // ========================================================================
    Message: {
      contentBg: '#ffffff',       // 背景
      borderRadius: 8,            // 圆角 (md)
    },

    // ========================================================================
    // Progress 组件
    // ========================================================================
    Progress: {
      defaultColor: '#6366f1',    // 默认颜色 (Primary 500)
      remainingColor: '#e9ecef',  // 剩余颜色 (Gray 200)
    },

    // ========================================================================
    // Spin 组件
    // ========================================================================
    Spin: {
      colorPrimary: '#6366f1',    // 主色 (Primary 500)
    },

    // ========================================================================
    // Layout 组件
    // ========================================================================
    Layout: {
      headerBg: '#ffffff',        // Header 背景
      headerHeight: 64,           // Header 高度
      headerPadding: '0 32px',    // Header 内边距
      siderBg: '#ffffff',         // Sider 背景
      bodyBg: '#fafbfc',          // Body 背景 (Gray 50)
      footerBg: '#ffffff',        // Footer 背景
      footerPadding: '24px 32px', // Footer 内边距
    },

    // ========================================================================
    // Menu 组件
    // ========================================================================
    Menu: {
      itemBg: 'transparent',      // 菜单项背景
      itemColor: '#495057',       // 菜单项文字颜色 (Gray 700)
      itemHoverBg: '#f8f9fa',     // 菜单项悬停背景 (Gray 100)
      itemHoverColor: '#212529',  // 菜单项悬停文字颜色 (Gray 900)
      itemSelectedBg: '#eef2ff',  // 菜单项选中背景 (Primary 50)
      itemSelectedColor: '#4338ca', // 菜单项选中文字颜色 (Primary 700)
      itemBorderRadius: 8,        // 菜单项圆角
      itemMarginBlock: 4,         // 菜单项垂直间距
      itemMarginInline: 12,       // 菜单项水平间距
      itemPaddingInline: 12,      // 菜单项内边距
    },
  },
};

export default antdTheme;
