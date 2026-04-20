import { Form, Input } from 'antd';
import { cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { renderWithAntd, screen, waitFor } from '../../test/utils/render';
import GuidedFormItem from './GuidedFormItem';
import type { FieldGuide } from './types';

const endpointGuide: FieldGuide = {
  whatToEnter: '填写 Kubernetes API Server 的完整 HTTPS 地址。',
  purpose: '平台会用这个地址发起连接验证并拉取集群信息。',
  example: 'https://api.k8s.example.com:6443',
  impact: '填错后连接测试会失败，集群无法导入。',
};

afterEach(() => {
  cleanup();
});

describe('GuidedFormItem', () => {
  it('shows and hides the guide card on focus transitions', async () => {
    const user = userEvent.setup();

    renderWithAntd(
      <Form layout="vertical">
        <GuidedFormItem name="endpoint" label="API Server" guide={endpointGuide}>
          <Input />
        </GuidedFormItem>
      </Form>,
    );

    expect(screen.queryByText('这里填什么')).not.toBeInTheDocument();

    await user.click(screen.getByLabelText('API Server'));

    expect(screen.getByText('这里填什么')).toBeInTheDocument();
    expect(screen.getByText('填写 Kubernetes API Server 的完整 HTTPS 地址。')).toBeInTheDocument();
    expect(screen.getByText('这个值是干嘛的')).toBeInTheDocument();
    expect(screen.getByText('推荐示例')).toBeInTheDocument();
    expect(screen.getByText('填错会怎样')).toBeInTheDocument();

    await user.tab();

    await waitFor(() => {
      expect(screen.queryByText('填写 Kubernetes API Server 的完整 HTTPS 地址。')).not.toBeInTheDocument();
    });
  });

  it('renders existing extra copy below the guide card while focused', async () => {
    const user = userEvent.setup();

    renderWithAntd(
      <Form layout="vertical">
        <GuidedFormItem
          name="endpoint"
          label="API Server"
          guide={endpointGuide}
          extra="例如: https://api.k8s.example.com:6443"
        >
          <Input />
        </GuidedFormItem>
      </Form>,
    );

    await user.click(screen.getByLabelText('API Server'));

    expect(screen.getByText('例如: https://api.k8s.example.com:6443')).toBeInTheDocument();
  });

  it('falls back to plain Form.Item behavior when guide is undefined', async () => {
    const user = userEvent.setup();

    renderWithAntd(
      <Form layout="vertical">
        <GuidedFormItem name="plain-field" label="普通字段">
          <Input />
        </GuidedFormItem>
      </Form>,
    );

    await user.click(screen.getByLabelText('普通字段'));

    expect(screen.queryByText('这里填什么')).not.toBeInTheDocument();
  });

  it('preserves child focus handlers', async () => {
    const user = userEvent.setup();
    const handleFocus = vi.fn();
    const handleBlur = vi.fn();

    renderWithAntd(
      <Form layout="vertical">
        <GuidedFormItem name="endpoint" label="API Server" guide={endpointGuide}>
          <Input onFocus={handleFocus} onBlur={handleBlur} />
        </GuidedFormItem>
        <Form.Item name="another" label="另一个字段">
          <Input />
        </Form.Item>
      </Form>,
    );

    await user.click(screen.getByLabelText('API Server'));

    expect(handleFocus).toHaveBeenCalledTimes(1);

    await user.click(screen.getByLabelText('另一个字段'));

    expect(handleBlur).toHaveBeenCalledTimes(1);
  });
});
