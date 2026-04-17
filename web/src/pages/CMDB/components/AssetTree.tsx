import React, { useState, useEffect } from 'react';
import { Tree, Input, Spin, Empty, message } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { useCMDBTree } from '../../../hooks/useCMDB';
import { cmdbApi } from '../../../api/modules/cmdb';
import type { CMDBTreeNode } from '../../../api/modules/cmdb';

interface AssetTreeProps {
  onSelect: (ciId: string | null) => void;
}

const { Search } = Input;

const AssetTree: React.FC<AssetTreeProps> = ({ onSelect }) => {
  const [treeData, setTreeData] = useState<DataNode[]>([]);
  const { data: rootNodes, loading, error } = useCMDBTree();

  useEffect(() => {
    if (rootNodes?.data) {
      setTreeData(mapNodesToDataNodes(rootNodes.data));
    }
  }, [rootNodes]);

  const mapNodesToDataNodes = (nodes: CMDBTreeNode[]): DataNode[] => {
    return nodes.map((node) => ({
      key: node.id,
      title: node.name,
      isLeaf: node.isLeaf,
      children: node.children ? mapNodesToDataNodes(node.children) : undefined,
    }));
  };

  const updateTreeData = (list: DataNode[], key: React.Key, children: DataNode[]): DataNode[] => {
    return list.map((node) => {
      if (node.key === key) {
        return {
          ...node,
          children,
        };
      }
      if (node.children) {
        return {
          ...node,
          children: updateTreeData(node.children, key, children),
        };
      }
      return node;
    });
  };

  const onLoadData = async ({ key, children }: any) => {
    if (children) {
      return;
    }
    try {
      const res = await cmdbApi.getTree({ parentId: Number(key) });
      if (res.data) {
        const newNodes = mapNodesToDataNodes(res.data);
        setTreeData((origin) => updateTreeData(origin, key, newNodes));
      }
    } catch (err) {
      message.error('加载子节点失败');
    }
  };

  const handleSelect = (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      onSelect(String(selectedKeys[0]));
    } else {
      onSelect(null);
    }
  };

  if (error) {
    return <Empty description="加载数据失败" />;
  }

  return (
    <div style={{ padding: '0 8px' }}>
      <Search
        placeholder="搜索资产..."
        style={{ marginBottom: 8 }}
        onSearch={(value) => console.log('Search:', value)}
        allowClear
      />
      {loading && treeData.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 20 }}>
          <Spin description="加载中..." />
        </div>
      ) : treeData.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无资产" />
      ) : (
        <Tree
          loadData={onLoadData}
          treeData={treeData}
          onSelect={handleSelect}
          showLine={{ showLeafIcon: false }}
          blockNode
        />
      )}
    </div>
  );
};

export default AssetTree;
