import React from 'react';
import { Modal, Divider, Tag } from 'antd';
import Kbd from '../Kbd';
import './KeyboardShortcutsHelp.css';

interface KeyboardShortcutsHelpProps {
  open: boolean;
  onClose: () => void;
}

interface ShortcutGroup {
  title: string;
  shortcuts: Array<{
    keys: string[];
    description: string;
  }>;
}

const KeyboardShortcutsHelp: React.FC<KeyboardShortcutsHelpProps> = ({ open, onClose }) => {
  const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
  const modKey = isMac ? '⌘' : 'Ctrl';

  const shortcutGroups: ShortcutGroup[] = [
    {
      title: '全局快捷键',
      shortcuts: [
        { keys: [modKey, 'K'], description: '打开命令面板' },
        { keys: ['/'], description: '聚焦搜索框' },
        { keys: ['Esc'], description: '关闭弹窗/取消操作' },
        { keys: ['?'], description: '打开快捷键帮助' },
      ],
    },
    {
      title: '导航快捷键',
      shortcuts: [
        { keys: ['g', 'h'], description: '跳转到首页' },
        { keys: ['g', 's'], description: '跳转到服务管理' },
        { keys: ['g', 'd'], description: '跳转到部署管理' },
        { keys: ['g', 'm'], description: '跳转到监控中心' },
        { keys: ['g', 'c'], description: '跳转到配置中心' },
        { keys: ['g', 't'], description: '跳转到任务中心' },
      ],
    },
    {
      title: '列表导航',
      shortcuts: [
        { keys: ['j'], description: '向下移动' },
        { keys: ['k'], description: '向上移动' },
        { keys: ['Enter'], description: '选择当前项' },
        { keys: ['Space'], description: '切换选中状态' },
      ],
    },
  ];

  return (
    <Modal
      title={
        <div className="flex items-center gap-2">
          <span className="text-lg font-semibold">键盘快捷键</span>
          <Tag color="blue">提升效率</Tag>
        </div>
      }
      open={open}
      onCancel={onClose}
      footer={null}
      width={640}
      className="keyboard-shortcuts-modal"
    >
      <div className="keyboard-shortcuts-content">
        {shortcutGroups.map((group, groupIndex) => (
          <div key={groupIndex} className="shortcut-group">
            <h3 className="shortcut-group-title">{group.title}</h3>
            <div className="shortcut-list">
              {group.shortcuts.map((shortcut, index) => (
                <div key={index} className="shortcut-item">
                  <div className="shortcut-keys">
                    {shortcut.keys.map((key, keyIndex) => (
                      <React.Fragment key={keyIndex}>
                        <Kbd>{key}</Kbd>
                        {keyIndex < shortcut.keys.length - 1 && (
                          <span className="shortcut-separator">+</span>
                        )}
                      </React.Fragment>
                    ))}
                  </div>
                  <div className="shortcut-description">{shortcut.description}</div>
                </div>
              ))}
            </div>
            {groupIndex < shortcutGroups.length - 1 && <Divider />}
          </div>
        ))}

        <div className="shortcut-footer">
          <p className="text-sm text-gray-500">
            💡 提示: 按 <Kbd>?</Kbd> 可随时打开此帮助
          </p>
        </div>
      </div>
    </Modal>
  );
};

export default KeyboardShortcutsHelp;
