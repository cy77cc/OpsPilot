import React from 'react';
import { Input, Radio, Space } from 'antd';

export type ScopeValue = {
  scope: 'global' | 'project';
  projectId?: string;
};

type ScopeSelectorProps = {
  value: ScopeValue;
  onChange: (next: ScopeValue) => void;
  disabled?: boolean;
};

const ScopeSelector: React.FC<ScopeSelectorProps> = ({ value, onChange, disabled }) => (
  <Space size={8}>
    <Radio.Group
      optionType="button"
      value={value.scope}
      disabled={disabled}
      onChange={(event) => {
        onChange({
          scope: event.target.value,
          projectId: value.projectId,
        });
      }}
      options={[
        { label: '全局', value: 'global' },
        { label: '项目', value: 'project' },
      ]}
    />
    {value.scope === 'project' ? (
      <Input
        placeholder="项目ID"
        value={value.projectId}
        disabled={disabled}
        style={{ width: 140 }}
        onChange={(event) => {
          onChange({
            scope: 'project',
            projectId: event.target.value,
          });
        }}
      />
    ) : null}
  </Space>
);

export default ScopeSelector;
