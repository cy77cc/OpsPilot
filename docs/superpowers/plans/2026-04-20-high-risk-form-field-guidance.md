# High-Risk Form Field Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a reusable focus-driven guidance card for high-risk form fields and wire it into the first three approved pages: cluster import, monitoring channels, and AI model settings.

**Architecture:** Add a small shared `FormGuidance` component set that wraps Ant Design `Form.Item` and injects a focus-aware guidance card through the `extra` slot. Keep guidance copy local to each page in adjacent `*FieldGuides.ts` files so every page owns its domain-specific wording while all pages share one interaction pattern and one test surface.

**Tech Stack:** React 19, TypeScript, Ant Design 6, Tailwind utility classes, Vitest, Testing Library

---

## File Map

- Create: `web/src/components/FormGuidance/types.ts`
  - Shared `FieldGuide` schema for all high-risk field copy.
- Create: `web/src/components/FormGuidance/FieldGuideCard.tsx`
  - Stateless renderer for the four-part guide card.
- Create: `web/src/components/FormGuidance/GuidedFormItem.tsx`
  - Focus-aware `Form.Item` wrapper that shows the card only while the field is active.
- Create: `web/src/components/FormGuidance/index.ts`
  - Barrel export used by page integrations.
- Create: `web/src/components/FormGuidance/GuidedFormItem.test.tsx`
  - Shared component tests for focus, blur, extra-copy composition, and graceful downgrade.
- Create: `web/src/pages/Deployment/Infrastructure/clusterImportFieldGuides.ts`
  - Chinese guidance copy for cluster import’s high-risk auth fields.
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx`
  - Swap selected `Form.Item` blocks to `GuidedFormItem`; replace the raw TLS checkbox with Ant Design `Checkbox`.
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx`
  - Add page-level guidance tests for certificate endpoint focus and insecure TLS focus.
- Create: `web/src/pages/Monitor/channelFieldGuides.ts`
  - Chinese guidance copy for monitoring channel provider, target, and JSON config fields.
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
  - Apply `GuidedFormItem` to test-send, create, and edit channel forms.
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
  - Add page-level guidance coverage for the create-channel modal.
- Create: `web/src/pages/Settings/aiModelFieldGuides.ts`
  - Chinese guidance copy for provider, model, base URL, API key, and temperature.
- Modify: `web/src/pages/Settings/AIModelSettingsPage.tsx`
  - Apply `GuidedFormItem` to the approved high-risk fields in the drawer form.
- Create: `web/src/pages/Settings/AIModelSettingsPage.test.tsx`
  - Add a focused smoke test for the new drawer guidance.

## Task 1: Build the Shared Form Guidance Primitive

**Files:**
- Create: `web/src/components/FormGuidance/types.ts`
- Create: `web/src/components/FormGuidance/FieldGuideCard.tsx`
- Create: `web/src/components/FormGuidance/GuidedFormItem.tsx`
- Create: `web/src/components/FormGuidance/index.ts`
- Test: `web/src/components/FormGuidance/GuidedFormItem.test.tsx`

- [ ] **Step 1: Write the failing shared-component tests**

```tsx
// web/src/components/FormGuidance/GuidedFormItem.test.tsx
import { Form, Input } from 'antd';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { renderWithAntd, screen, waitFor } from '../../test/utils/render';
import GuidedFormItem from './GuidedFormItem';
import type { FieldGuide } from './types';

const endpointGuide: FieldGuide = {
  whatToEnter: '填写 Kubernetes API Server 的完整 HTTPS 地址。',
  purpose: '平台会用这个地址发起连接验证并拉取集群信息。',
  example: 'https://api.k8s.example.com:6443',
  impact: '填错后连接测试会失败，集群无法导入。',
};

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

    renderWithAntd(
      <Form layout="vertical">
        <GuidedFormItem name="endpoint" label="API Server" guide={endpointGuide}>
          <Input onFocus={handleFocus} />
        </GuidedFormItem>
      </Form>,
    );

    await user.click(screen.getByLabelText('API Server'));

    expect(handleFocus).toHaveBeenCalledTimes(1);
  });
});
```

- [ ] **Step 2: Run the shared-component test to verify it fails**

Run: `cd web && npm run test:run -- src/components/FormGuidance/GuidedFormItem.test.tsx`

Expected: FAIL because `GuidedFormItem.tsx` and `FieldGuideCard.tsx` do not exist yet.

- [ ] **Step 3: Write the minimal shared implementation**

```ts
// web/src/components/FormGuidance/types.ts
export type FieldGuide = {
  whatToEnter: string;
  purpose: string;
  example?: string;
  impact?: string;
  whenRequired?: string;
  formatNotes?: string;
};
```

```tsx
// web/src/components/FormGuidance/FieldGuideCard.tsx
import React from 'react';
import type { FieldGuide } from './types';

type FieldGuideRowProps = {
  label: string;
  value: string;
};

const FieldGuideRow: React.FC<FieldGuideRowProps> = ({ label, value }) => (
  <div className="space-y-1">
    <div className="text-xs font-semibold tracking-wide text-emerald-700">{label}</div>
    <div className="text-sm leading-6 text-slate-700">{value}</div>
  </div>
);

type FieldGuideCardProps = {
  guide: FieldGuide;
};

const FieldGuideCard: React.FC<FieldGuideCardProps> = ({ guide }) => {
  const rows = [
    { label: '这里填什么', value: guide.whatToEnter },
    { label: '这个值是干嘛的', value: guide.purpose },
    { label: '推荐示例', value: guide.example },
    { label: '填错会怎样', value: guide.impact },
    { label: '什么时候必填', value: guide.whenRequired },
    { label: '格式要求', value: guide.formatNotes },
  ].filter((row): row is { label: string; value: string } => Boolean(row.value && row.value.trim()));

  return (
    <div
      data-testid="field-guide-card"
      className="rounded-xl border border-emerald-200 bg-emerald-50/80 p-3 shadow-sm"
    >
      <div className="space-y-3">
        {rows.map((row) => (
          <FieldGuideRow key={row.label} label={row.label} value={row.value} />
        ))}
      </div>
    </div>
  );
};

export default FieldGuideCard;
```

```tsx
// web/src/components/FormGuidance/GuidedFormItem.tsx
import React from 'react';
import { Form } from 'antd';
import type { FormItemProps } from 'antd';
import FieldGuideCard from './FieldGuideCard';
import type { FieldGuide } from './types';

type FocusableChildProps = {
  onFocus?: React.FocusEventHandler<HTMLElement>;
  onBlur?: React.FocusEventHandler<HTMLElement>;
};

export interface GuidedFormItemProps extends Omit<FormItemProps, 'children'> {
  guide?: FieldGuide;
  children: React.ReactElement<FocusableChildProps>;
}

const callFocusHandler = (
  handler: React.FocusEventHandler<HTMLElement> | undefined,
  event: React.FocusEvent<HTMLElement>,
) => {
  if (handler) handler(event);
};

const GuidedFormItem: React.FC<GuidedFormItemProps> = ({ guide, extra, children, ...formItemProps }) => {
  const [isFocused, setIsFocused] = React.useState(false);
  const child = children as React.ReactElement<FocusableChildProps>;

  const mergedExtra =
    guide && isFocused ? (
      <div className="space-y-2">
        <FieldGuideCard guide={guide} />
        {extra ? <div>{extra}</div> : null}
      </div>
    ) : (
      extra
    );

  const enhancedChild = React.cloneElement(child, {
    onFocus: (event: React.FocusEvent<HTMLElement>) => {
      setIsFocused(true);
      callFocusHandler(child.props.onFocus, event);
    },
    onBlur: (event: React.FocusEvent<HTMLElement>) => {
      setIsFocused(false);
      callFocusHandler(child.props.onBlur, event);
    },
  });

  return (
    <Form.Item {...formItemProps} extra={mergedExtra}>
      {enhancedChild}
    </Form.Item>
  );
};

export default GuidedFormItem;
```

```ts
// web/src/components/FormGuidance/index.ts
export type { FieldGuide } from './types';
export { default as FieldGuideCard } from './FieldGuideCard';
export { default as GuidedFormItem } from './GuidedFormItem';
```

- [ ] **Step 4: Run the shared-component test to verify it passes**

Run: `cd web && npm run test:run -- src/components/FormGuidance/GuidedFormItem.test.tsx`

Expected: PASS for all four `GuidedFormItem` tests.

- [ ] **Step 5: Commit the shared component**

```bash
git add \
  web/src/components/FormGuidance/types.ts \
  web/src/components/FormGuidance/FieldGuideCard.tsx \
  web/src/components/FormGuidance/GuidedFormItem.tsx \
  web/src/components/FormGuidance/index.ts \
  web/src/components/FormGuidance/GuidedFormItem.test.tsx
git commit -m "feat(web): add guided form item"
```

## Task 2: Integrate Cluster Import Guidance

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/clusterImportFieldGuides.ts`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx`
- Test: `web/src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx`

- [ ] **Step 1: Add failing cluster-import guidance tests**

```tsx
// append to web/src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx
  it('shows endpoint guidance when certificate auth fields receive focus', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('radio', { name: /API 地址 \+ 证书/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    const endpointInput = screen.getByLabelText('API Server 地址');
    fireEvent.focus(endpointInput);

    expect(screen.getByText('这里填什么')).toBeInTheDocument();
    expect(screen.getByText('填写目标集群 Kubernetes API Server 的完整 HTTPS 地址。')).toBeInTheDocument();

    fireEvent.blur(endpointInput);

    await waitFor(() => {
      expect(screen.queryByText('填写目标集群 Kubernetes API Server 的完整 HTTPS 地址。')).not.toBeInTheDocument();
    });
  });

  it('shows insecure TLS guidance when the token checkbox receives focus', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ClusterImportWizard />);

    await user.type(screen.getByLabelText('集群名称'), 'prod-k8s');
    await user.click(screen.getByRole('button', { name: '下一步' }));
    await user.click(screen.getByRole('radio', { name: /ServiceAccount Token/ }));
    await user.click(screen.getByRole('button', { name: '下一步' }));

    const skipTlsCheckbox = screen.getByRole('checkbox');
    fireEvent.focus(skipTlsCheckbox);

    expect(screen.getByText('只在测试环境或临时排障时启用。')).toBeInTheDocument();
    expect(screen.getByText('开启后虽然可能绕过证书问题，但也会放大中间人攻击风险。')).toBeInTheDocument();

    fireEvent.blur(skipTlsCheckbox);

    await waitFor(() => {
      expect(screen.queryByText('只在测试环境或临时排障时启用。')).not.toBeInTheDocument();
    });
  });
```

- [ ] **Step 2: Run the cluster-import page test to verify it fails**

Run: `cd web && npm run test:run -- src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx`

Expected: FAIL because the new focus assertions cannot find the guidance copy.

- [ ] **Step 3: Add the page guide map and wire the wizard fields**

```ts
// web/src/pages/Deployment/Infrastructure/clusterImportFieldGuides.ts
import type { FieldGuide } from '../../../components/FormGuidance';

type ClusterImportGuideKey =
  | 'kubeconfig'
  | 'endpoint'
  | 'ca_cert'
  | 'cert'
  | 'key'
  | 'token'
  | 'skip_tls_verify';

export const clusterImportFieldGuides: Record<ClusterImportGuideKey, FieldGuide> = {
  kubeconfig: {
    whatToEnter: '粘贴完整的 kubeconfig 内容，或上传导出的配置文件。',
    purpose: '平台会从中读取集群地址、证书和上下文，用来验证连接并导入集群。',
    example: 'apiVersion: v1 ... current-context: production-cluster',
    impact: '内容不完整、上下文错误或权限不足时，连接测试会失败。',
  },
  endpoint: {
    whatToEnter: '填写目标集群 Kubernetes API Server 的完整 HTTPS 地址。',
    purpose: '平台会把所有连接验证和后续同步请求发往这个地址。',
    example: 'https://api.k8s.example.com:6443',
    impact: '地址写错、协议不对或端口不可达时，连接测试和导入都会失败。',
  },
  ca_cert: {
    whatToEnter: '填写用来校验 API Server 身份的 CA 证书内容。',
    purpose: '平台会用它验证服务端证书是否可信，避免连到错误集群。',
    example: '-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----',
    impact: '证书缺失、格式错误或与目标集群不匹配时，TLS 握手会失败。',
  },
  cert: {
    whatToEnter: '填写有权访问该集群的客户端证书内容。',
    purpose: '证书认证模式下，平台会用它代表当前用户或系统身份访问集群。',
    example: '-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----',
    impact: '客户端证书无效、过期或权限不足时，连接测试会返回认证失败。',
  },
  key: {
    whatToEnter: '填写与客户端证书配套的私钥内容。',
    purpose: '平台会用它和客户端证书配对完成双向 TLS 认证。',
    example: '-----BEGIN RSA PRIVATE KEY----- ... -----END RSA PRIVATE KEY-----',
    impact: '私钥与证书不匹配或格式损坏时，证书认证无法建立连接。',
  },
  token: {
    whatToEnter: '填写可访问目标集群的 ServiceAccount Bearer Token。',
    purpose: 'Token 认证模式下，平台会用它向 API Server 发起授权请求。',
    example: 'eyJhbGciOiJSUzI1NiIsImtpZCI6Ii...',
    impact: 'Token 失效、权限不足或粘贴错误时，连接测试会返回 401/403。',
  },
  skip_tls_verify: {
    whatToEnter: '只在测试环境或临时排障时启用。',
    purpose: '开启后平台会跳过 API Server 证书校验，作为缺少可信 CA 时的临时兜底。',
    example: '仅在你明确确认目标 API Server 地址可信时勾选。',
    impact: '开启后虽然可能绕过证书问题，但也会放大中间人攻击风险。',
  },
};
```

```tsx
// imports inside web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx
import {
  Steps, Form, Input, Button, Card, Space, message, Upload, Alert, Result,
  Radio, Spin, Descriptions, Tag, Divider, Typography, Checkbox
} from 'antd';
import { GuidedFormItem } from '../../../components/FormGuidance';
import { clusterImportFieldGuides } from './clusterImportFieldGuides';
```

```tsx
// replace the kubeconfig block inside renderKubeconfigForm()
<GuidedFormItem
  name="kubeconfig"
  label="Kubeconfig 内容"
  guide={clusterImportFieldGuides.kubeconfig}
  rules={[{ required: true, message: '请输入或上传 kubeconfig' }]}
>
  <TextArea
    rows={12}
    placeholder="粘贴 kubeconfig 内容，或点击下方按钮上传文件"
    style={{ fontFamily: 'monospace', fontSize: '12px' }}
  />
</GuidedFormItem>
```

```tsx
// replace the certificate fields inside renderCertificateForm()
<GuidedFormItem
  name="endpoint"
  label="API Server 地址"
  guide={clusterImportFieldGuides.endpoint}
  rules={[{ required: true, message: '请输入 API Server 地址' }]}
>
  <Input placeholder="https://api.k8s.example.com:6443" />
</GuidedFormItem>

<GuidedFormItem
  name="ca_cert"
  label="CA 证书"
  guide={clusterImportFieldGuides.ca_cert}
  rules={[{ required: true, message: '请输入 CA 证书' }]}
>
  <TextArea
    rows={4}
    placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
    style={{ fontFamily: 'monospace', fontSize: '12px' }}
  />
</GuidedFormItem>

<GuidedFormItem
  name="cert"
  label="客户端证书"
  guide={clusterImportFieldGuides.cert}
  rules={[{ required: true, message: '请输入客户端证书' }]}
  className="mt-4"
>
  <TextArea
    rows={4}
    placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
    style={{ fontFamily: 'monospace', fontSize: '12px' }}
  />
</GuidedFormItem>

<GuidedFormItem
  name="key"
  label="客户端私钥"
  guide={clusterImportFieldGuides.key}
  rules={[{ required: true, message: '请输入客户端私钥' }]}
  className="mt-4"
>
  <TextArea
    rows={4}
    placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
    style={{ fontFamily: 'monospace', fontSize: '12px' }}
  />
</GuidedFormItem>
```

```tsx
// replace the token fields inside renderTokenForm()
<GuidedFormItem
  name="endpoint"
  label="API Server 地址"
  guide={clusterImportFieldGuides.endpoint}
  rules={[{ required: true, message: '请输入 API Server 地址' }]}
>
  <Input placeholder="https://api.k8s.example.com:6443" />
</GuidedFormItem>

<GuidedFormItem
  name="ca_cert"
  label="CA 证书（可选）"
  guide={clusterImportFieldGuides.ca_cert}
>
  <TextArea
    rows={4}
    placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
    style={{ fontFamily: 'monospace', fontSize: '12px' }}
  />
</GuidedFormItem>

<GuidedFormItem
  name="token"
  label="Bearer Token"
  guide={clusterImportFieldGuides.token}
  rules={[{ required: true, message: '请输入 Token' }]}
  className="mt-4"
>
  <TextArea
    rows={4}
    placeholder="eyJhbGciOiJSUzI1NiIsImtpZCI6Ii..."
    style={{ fontFamily: 'monospace', fontSize: '12px' }}
  />
</GuidedFormItem>

<GuidedFormItem
  name="skip_tls_verify"
  guide={clusterImportFieldGuides.skip_tls_verify}
  valuePropName="checked"
  className="mt-2"
>
  <Checkbox>跳过 TLS 证书验证（不推荐）</Checkbox>
</GuidedFormItem>
```

- [ ] **Step 4: Run the cluster-import page test to verify it passes**

Run: `cd web && npm run test:run -- src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx`

Expected: PASS for the existing import-flow coverage plus the two new guidance tests.

- [ ] **Step 5: Commit the cluster-import integration**

```bash
git add \
  web/src/pages/Deployment/Infrastructure/clusterImportFieldGuides.ts \
  web/src/pages/Deployment/Infrastructure/ClusterImportWizard.tsx \
  web/src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx
git commit -m "feat(web): guide cluster import auth fields"
```

## Task 3: Integrate Monitoring Channel Guidance

**Files:**
- Create: `web/src/pages/Monitor/channelFieldGuides.ts`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Test: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`

- [ ] **Step 1: Add the failing monitoring-channel guidance test**

```tsx
// append to web/src/pages/Monitor/ChannelsConfigPage.test.tsx
  it('shows config JSON guidance in the create-channel modal', async () => {
    render(<ChannelsConfigPage />);
    await screen.findByText('Ops Webhook');

    fireEvent.click(screen.getByRole('button', { name: '新增渠道' }));

    const dialog = await screen.findByRole('dialog', { name: '新增渠道' });
    const configInput = within(dialog).getByLabelText('配置 JSON');

    fireEvent.focus(configInput);

    expect(within(dialog).getByText('这里填什么')).toBeInTheDocument();
    expect(within(dialog).getByText('这里填当前渠道 provider 需要的附加配置，必须是合法 JSON。')).toBeInTheDocument();

    fireEvent.blur(configInput);

    await waitFor(() => {
      expect(screen.queryByText('这里填当前渠道 provider 需要的附加配置，必须是合法 JSON。')).not.toBeInTheDocument();
    });
  });
```

- [ ] **Step 2: Run the monitoring-channel page test to verify it fails**

Run: `cd web && npm run test:run -- src/pages/Monitor/ChannelsConfigPage.test.tsx`

Expected: FAIL because the create modal has no guidance card yet.

- [ ] **Step 3: Add the channel guide map and apply `GuidedFormItem`**

```ts
// web/src/pages/Monitor/channelFieldGuides.ts
import type { FieldGuide } from '../../components/FormGuidance';

type ChannelGuideKey = 'provider' | 'target' | 'configJson';

export const channelFieldGuides: Record<ChannelGuideKey, FieldGuide> = {
  provider: {
    whatToEnter: '填写通知渠道类型的枚举值。',
    purpose: '平台会按这个 provider 解析目标地址、配置 JSON 和实际投递方式。',
    example: 'webhook / email / log',
    impact: 'provider 写错时，测试发送和真实告警投递都可能失败。',
  },
  target: {
    whatToEnter: '填写当前渠道的实际投递目标。',
    purpose: '不同 provider 会把这个值当作 webhook 地址、邮箱地址或其他接收端标识。',
    example: 'https://example.com/hook 或 ops@example.com',
    impact: '目标地址填错时，请求会打到错误位置，测试发送不会成功。',
  },
  configJson: {
    whatToEnter: '这里填当前渠道 provider 需要的附加配置，必须是合法 JSON。',
    purpose: '平台会把这段 JSON 作为 headers、鉴权信息或模板参数一起传给通知驱动。',
    example: '{"headers":{"X-Env":"prod"}}',
    impact: 'JSON 语法错误或字段名不对时，保存后测试发送也可能失败。',
  },
};
```

```tsx
// imports inside web/src/pages/Monitor/ChannelsConfigPage.tsx
import { GuidedFormItem } from '../../components/FormGuidance';
import { channelFieldGuides } from './channelFieldGuides';
```

```tsx
// replace the test-send form items
<GuidedFormItem
  label="Provider"
  name="provider"
  guide={channelFieldGuides.provider}
  rules={[{ required: true, message: '请选择 provider' }]}
>
  <Select
    options={[
      { label: 'Webhook', value: 'webhook' },
      { label: 'Log', value: 'log' },
      { label: 'Email', value: 'email' },
    ]}
  />
</GuidedFormItem>

<GuidedFormItem label="目标地址" name="target" guide={channelFieldGuides.target}>
  <Input placeholder="https://example.com/hook" />
</GuidedFormItem>

<GuidedFormItem label="配置 JSON" name="configJson" guide={channelFieldGuides.configJson}>
  <Input.TextArea rows={4} />
</GuidedFormItem>
```

```tsx
// replace the create modal form items
<Form.Item label="名称" name="channelName" rules={[{ required: true, message: '请输入名称' }]}>
  <Input placeholder="渠道名称" />
</Form.Item>

<GuidedFormItem
  label="Provider"
  name="channelProvider"
  guide={channelFieldGuides.provider}
  rules={[{ required: true, message: '请输入 provider' }]}
>
  <Input />
</GuidedFormItem>

<GuidedFormItem label="目标地址" name="channelTarget" guide={channelFieldGuides.target}>
  <Input />
</GuidedFormItem>

<GuidedFormItem label="配置 JSON" name="channelConfigJson" guide={channelFieldGuides.configJson}>
  <Input.TextArea rows={4} />
</GuidedFormItem>
```

```tsx
// replace the edit modal form items
<Form.Item label="名称" name="channelName" rules={[{ required: true, message: '请输入名称' }]}>
  <Input />
</Form.Item>

<GuidedFormItem
  label="Provider"
  name="channelProvider"
  guide={channelFieldGuides.provider}
  rules={[{ required: true, message: '请输入 provider' }]}
>
  <Input />
</GuidedFormItem>

<GuidedFormItem label="目标地址" name="channelTarget" guide={channelFieldGuides.target}>
  <Input />
</GuidedFormItem>

<GuidedFormItem label="配置 JSON" name="channelConfigJson" guide={channelFieldGuides.configJson}>
  <Input.TextArea rows={4} />
</GuidedFormItem>
```

- [ ] **Step 4: Run the monitoring-channel page test to verify it passes**

Run: `cd web && npm run test:run -- src/pages/Monitor/ChannelsConfigPage.test.tsx`

Expected: PASS for existing CRUD tests plus the new create-modal guidance assertion.

- [ ] **Step 5: Commit the monitoring-channel integration**

```bash
git add \
  web/src/pages/Monitor/channelFieldGuides.ts \
  web/src/pages/Monitor/ChannelsConfigPage.tsx \
  web/src/pages/Monitor/ChannelsConfigPage.test.tsx
git commit -m "feat(web): guide monitoring channel fields"
```

## Task 4: Integrate AI Model Settings Guidance

**Files:**
- Create: `web/src/pages/Settings/aiModelFieldGuides.ts`
- Modify: `web/src/pages/Settings/AIModelSettingsPage.tsx`
- Test: `web/src/pages/Settings/AIModelSettingsPage.test.tsx`

- [ ] **Step 1: Write the failing AI-model guidance test**

```tsx
// web/src/pages/Settings/AIModelSettingsPage.test.tsx
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { cleanup, fireEvent, renderWithProviders, screen, waitFor } from '../../test/utils/render';
import AIModelSettingsPage from './AIModelSettingsPage';

const mockApi = vi.hoisted(() => ({
  ai: {
    listAdminModels: vi.fn(),
    createAdminModel: vi.fn(),
    updateAdminModel: vi.fn(),
    setAdminDefaultModel: vi.fn(),
    deleteAdminModel: vi.fn(),
    previewAdminModelImport: vi.fn(),
    importAdminModels: vi.fn(),
  },
}));

vi.mock('../../api', () => ({
  Api: mockApi,
}));

describe('AIModelSettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApi.ai.listAdminModels.mockResolvedValue({ data: { list: [] } });
  });

  afterEach(() => {
    cleanup();
  });

  it('shows base URL guidance in the create drawer', async () => {
    const user = userEvent.setup();

    renderWithProviders(<AIModelSettingsPage />);
    await screen.findByText('AI 模型配置中心');

    await user.click(screen.getByRole('button', { name: '新增模型' }));

    const baseUrlInput = await screen.findByLabelText('Base URL');
    fireEvent.focus(baseUrlInput);

    expect(screen.getByText('这里填什么')).toBeInTheDocument();
    expect(screen.getByText('填写模型供应商实际提供的接口根地址。')).toBeInTheDocument();

    fireEvent.blur(baseUrlInput);

    await waitFor(() => {
      expect(screen.queryByText('填写模型供应商实际提供的接口根地址。')).not.toBeInTheDocument();
    });
  });
});
```

- [ ] **Step 2: Run the AI-model page test to verify it fails**

Run: `cd web && npm run test:run -- src/pages/Settings/AIModelSettingsPage.test.tsx`

Expected: FAIL because the drawer fields still use plain `Form.Item`.

- [ ] **Step 3: Add the AI-model guide map and wire the drawer form**

```ts
// web/src/pages/Settings/aiModelFieldGuides.ts
import type { FieldGuide } from '../../components/FormGuidance';

type AIModelGuideKey = 'provider' | 'model' | 'base_url' | 'api_key' | 'temperature';

export const aiModelFieldGuides: Record<AIModelGuideKey, FieldGuide> = {
  provider: {
    whatToEnter: '选择当前模型所属的供应商。',
    purpose: '平台会按供应商切换兼容协议、默认路由和后续调用方式。',
    example: 'Qwen / OpenAI / Ark / Ollama / MiniMax',
    impact: '供应商选错时，模型标识和 Base URL 即使正确，也可能因为协议不匹配而调用失败。',
  },
  model: {
    whatToEnter: '填写供应商侧真实可调用的模型标识。',
    purpose: '平台会把它作为请求参数发给上游模型服务。',
    example: 'qwen-max / gpt-4.1 / doubao-pro-32k',
    impact: '模型标识写错时，请求会返回模型不存在或路由失败。',
  },
  base_url: {
    whatToEnter: '填写模型供应商实际提供的接口根地址。',
    purpose: '平台会把所有对话和推理请求发到这个地址。',
    example: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    impact: '地址写错、缺协议或路径不兼容时，连通性和调用都会失败。',
  },
  api_key: {
    whatToEnter: '填写有权调用该模型的 API Key。',
    purpose: '平台会把它作为上游鉴权凭证，向模型供应商发起请求。',
    example: 'sk-xxxxxx',
    impact: 'Key 无效、复制不完整或配错环境时，模型保存后仍会调用失败。',
  },
  temperature: {
    whatToEnter: '填写 0 到 2 之间的采样温度。',
    purpose: '这个值会影响模型输出的随机性和稳定性。',
    example: '0.2 更稳、0.7 常用、1.2 更发散',
    impact: '值过高会让输出更不稳定，值过低则可能让回答过于保守。',
  },
};
```

```tsx
// imports inside web/src/pages/Settings/AIModelSettingsPage.tsx
import { GuidedFormItem } from '../../components/FormGuidance';
import { aiModelFieldGuides } from './aiModelFieldGuides';
```

```tsx
// replace the guided fields inside the drawer form
<Form.Item name="name" label="显示名称" rules={[{ required: true, message: '请输入模型显示名称' }]}>
  <Input placeholder="例如：Qwen3.5-Plus（生产）" />
</Form.Item>

<GuidedFormItem name="provider" label="供应商" guide={aiModelFieldGuides.provider} rules={[{ required: true }]}>
  <Select options={providerChoices} placeholder="请选择供应商" />
</GuidedFormItem>

<GuidedFormItem
  name="model"
  label="模型标识"
  guide={aiModelFieldGuides.model}
  rules={[{ required: true, message: '请输入模型标识' }]}
>
  <Input placeholder="qwen-max / doubao-pro / llama3" />
</GuidedFormItem>

<GuidedFormItem
  name="base_url"
  label="Base URL"
  guide={aiModelFieldGuides.base_url}
  rules={[{ required: true, message: '请输入 Base URL' }]}
>
  <Input placeholder="https://dashscope.aliyuncs.com/compatible-mode/v1" />
</GuidedFormItem>

<GuidedFormItem
  name="api_key"
  label={editing ? 'API Key（留空则不修改）' : 'API Key'}
  guide={aiModelFieldGuides.api_key}
  rules={editing ? [] : [{ required: true, message: '请输入 API Key' }]}
>
  <Input.Password placeholder={editing ? '留空表示保持原值' : '输入 API Key'} />
</GuidedFormItem>

<GuidedFormItem name="temperature" label="Temperature" guide={aiModelFieldGuides.temperature}>
  <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
</GuidedFormItem>
```

- [ ] **Step 4: Run the AI-model page test to verify it passes**

Run: `cd web && npm run test:run -- src/pages/Settings/AIModelSettingsPage.test.tsx`

Expected: PASS for the new drawer guidance smoke test.

- [ ] **Step 5: Commit the AI-model integration**

```bash
git add \
  web/src/pages/Settings/aiModelFieldGuides.ts \
  web/src/pages/Settings/AIModelSettingsPage.tsx \
  web/src/pages/Settings/AIModelSettingsPage.test.tsx
git commit -m "feat(web): guide AI model settings fields"
```

## Final Verification Matrix

- [ ] Run the focused guidance suite

Run:

```bash
cd web
npm run test:run -- \
  src/components/FormGuidance/GuidedFormItem.test.tsx \
  src/pages/Deployment/Infrastructure/ClusterImportWizard.test.tsx \
  src/pages/Monitor/ChannelsConfigPage.test.tsx \
  src/pages/Settings/AIModelSettingsPage.test.tsx
```

Expected: PASS across the shared component plus all three first-batch pages.

- [ ] Run the frontend production build

Run:

```bash
cd web
npm run build
```

Expected: `vite build` completes without TypeScript or bundling errors.

- [ ] Manual smoke-check the approved interaction

Run:

```bash
cd web
npm run dev
```

Then verify in the browser:

- Cluster import: focusing `API Server 地址`, `CA 证书`, `Bearer Token`, and `跳过 TLS 证书验证（不推荐）` shows a card only while the field is active.
- Monitoring channels: focusing `Provider`, `目标地址`, and `配置 JSON` in the test-send card and create/edit modals shows the correct copy without permanently expanding the form.
- AI model settings: focusing `供应商`, `模型标识`, `Base URL`, `API Key`, and `Temperature` inside the drawer shows the card and leaves ordinary fields unchanged.
- Existing validation errors still appear in Ant Design’s standard location after blur or submit.

## Spec Coverage Check

- Reusable shared primitive: covered by Task 1.
- First-batch page rollout: covered by Tasks 2, 3, and 4.
- Chinese-only page-local copy: covered by the three `*FieldGuides.ts` files.
- Focus-only behavior with no global side effects: covered by Task 1’s component tests and the per-page focus tests.
- Preserve current validation and submission flows: covered by re-running the existing page tests in Tasks 2 and 3, and by the build + smoke checks in the verification matrix.
