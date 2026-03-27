import React, { useState, useEffect, useCallback } from 'react';
import { Command } from 'cmdk';
import { useNavigate } from 'react-router-dom';
import {
  HomeOutlined,
  AppstoreOutlined,
  CloudUploadOutlined,
  DesktopOutlined,
  BarChartOutlined,
  SettingOutlined,
  SearchOutlined,
  RightOutlined,
} from '@ant-design/icons';
import './CommandPalette.css';

interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const CommandPalette: React.FC<CommandPaletteProps> = ({ open, onOpenChange }) => {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [pages, setPages] = useState<string[]>([]);
  const page = pages[pages.length - 1];

  // Close on Escape
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onOpenChange(false);
        setPages([]);
      }
    };

    document.addEventListener('keydown', down);
    return () => document.removeEventListener('keydown', down);
  }, [onOpenChange]);

  // Reset search when opening
  useEffect(() => {
    if (open) {
      setSearch('');
    }
  }, [open]);

  const handleNavigate = useCallback(
    (path: string) => {
      navigate(path);
      onOpenChange(false);
      setPages([]);
    },
    [navigate, onOpenChange]
  );

  if (!open) return null;

  return (
    <div className="command-palette-overlay" onClick={() => onOpenChange(false)}>
      <div className="command-palette-container" onClick={(e) => e.stopPropagation()}>
        <Command label="Command Menu" shouldFilter={true}>
          <div className="command-palette-header">
            <SearchOutlined className="command-palette-search-icon" />
            <Command.Input
              value={search}
              onValueChange={setSearch}
              placeholder="搜索命令或页面..."
              className="command-palette-input"
            />
          </div>

          <Command.List className="command-palette-list">
            <Command.Empty className="command-palette-empty">
              没有找到相关命令
            </Command.Empty>

            {!page && (
              <>
                <Command.Group heading="导航" className="command-palette-group">
                  <Command.Item
                    onSelect={() => handleNavigate('/')}
                    className="command-palette-item"
                  >
                    <HomeOutlined className="command-palette-item-icon" />
                    <span>仪表盘</span>
                    <span className="command-palette-item-shortcut">g h</span>
                  </Command.Item>

                  <Command.Item
                    onSelect={() => handleNavigate('/services')}
                    className="command-palette-item"
                  >
                    <AppstoreOutlined className="command-palette-item-icon" />
                    <span>服务管理</span>
                    <span className="command-palette-item-shortcut">g s</span>
                  </Command.Item>

                  <Command.Item
                    onSelect={() => handleNavigate('/deployment')}
                    className="command-palette-item"
                  >
                    <CloudUploadOutlined className="command-palette-item-icon" />
                    <span>部署管理</span>
                    <span className="command-palette-item-shortcut">g d</span>
                  </Command.Item>

                  <Command.Item
                    onSelect={() => handleNavigate('/hosts')}
                    className="command-palette-item"
                  >
                    <DesktopOutlined className="command-palette-item-icon" />
                    <span>主机管理</span>
                    <span className="command-palette-item-shortcut">g h</span>
                  </Command.Item>

                  <Command.Item
                    onSelect={() => handleNavigate('/monitoring')}
                    className="command-palette-item"
                  >
                    <BarChartOutlined className="command-palette-item-icon" />
                    <span>监控中心</span>
                    <span className="command-palette-item-shortcut">g m</span>
                  </Command.Item>

                  <Command.Item
                    onSelect={() => handleNavigate('/config')}
                    className="command-palette-item"
                  >
                    <SettingOutlined className="command-palette-item-icon" />
                    <span>配置中心</span>
                    <span className="command-palette-item-shortcut">g c</span>
                  </Command.Item>
                </Command.Group>

                <Command.Group heading="操作" className="command-palette-group">
                  <Command.Item
                    onSelect={() => setPages([...pages, 'create'])}
                    className="command-palette-item"
                  >
                    <span>创建资源</span>
                    <RightOutlined className="command-palette-item-arrow" />
                  </Command.Item>
                </Command.Group>
              </>
            )}

            {page === 'create' && (
              <Command.Group heading="创建资源" className="command-palette-group">
                <Command.Item
                  onSelect={() => handleNavigate('/services/provision')}
                  className="command-palette-item"
                >
                  <AppstoreOutlined className="command-palette-item-icon" />
                  <span>创建服务</span>
                </Command.Item>

                <Command.Item
                  onSelect={() => handleNavigate('/deployment/create')}
                  className="command-palette-item"
                >
                  <CloudUploadOutlined className="command-palette-item-icon" />
                  <span>创建部署</span>
                </Command.Item>

                <Command.Item
                  onSelect={() => handleNavigate('/hosts/onboarding')}
                  className="command-palette-item"
                >
                  <DesktopOutlined className="command-palette-item-icon" />
                  <span>添加主机</span>
                </Command.Item>
              </Command.Group>
            )}
          </Command.List>

          {pages.length > 0 && (
            <div className="command-palette-footer">
              <button
                onClick={() => setPages(pages.slice(0, -1))}
                className="command-palette-back-button"
              >
                返回
              </button>
            </div>
          )}
        </Command>
      </div>
    </div>
  );
};

export default CommandPalette;
