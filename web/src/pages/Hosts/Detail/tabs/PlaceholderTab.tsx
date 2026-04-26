import React from 'react';

const PlaceholderTab: React.FC<{ name: string }> = ({ name }) => {
  return (
    <div className="py-8 text-center border border-dashed border-gray-300 rounded-lg bg-gray-50">
      <div className="text-gray-400 mb-2">
        <span className="text-4xl">Construction</span>
      </div>
      <div className="text-lg font-medium text-gray-600">{name} 模块正在开发中</div>
      <div className="text-sm text-gray-400 mt-1">该模块将独立实现，目前仅供布局展示</div>
    </div>
  );
};

export default PlaceholderTab;
