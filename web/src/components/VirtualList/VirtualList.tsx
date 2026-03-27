import React from 'react';
// @ts-ignore - react-window types issue
import { FixedSizeList as List } from 'react-window';
// @ts-ignore - react-virtualized-auto-sizer types issue
import { AutoSizer } from 'react-virtualized-auto-sizer';

interface VirtualListProps<T> {
  items: T[];
  itemHeight: number;
  renderItem: (item: T, index: number) => React.ReactNode;
  className?: string;
  overscanCount?: number;
}

/**
 * 虚拟滚动列表组件
 *
 * 使用 react-window 实现虚拟滚动，优化大列表性能
 *
 * 特性:
 * - 只渲染可见区域的项目
 * - 自动计算容器尺寸
 * - 支持自定义项目高度
 * - 支持预渲染（overscan）
 */
function VirtualList<T>({
  items,
  itemHeight,
  renderItem,
  className = '',
  overscanCount = 5,
}: VirtualListProps<T>) {
  const Row = ({ index, style }: any) => (
    <div style={style}>{renderItem(items[index], index)}</div>
  );

  return (
    <div className={className} style={{ height: '100%', width: '100%' }}>
      {/* @ts-ignore */}
      <AutoSizer>
        {({ height, width }: any) => (
          <List
            height={height}
            itemCount={items.length}
            itemSize={itemHeight}
            width={width}
            overscanCount={overscanCount}
          >
            {Row}
          </List>
        )}
      </AutoSizer>
    </div>
  );
}

export default VirtualList;
