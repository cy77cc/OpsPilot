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
