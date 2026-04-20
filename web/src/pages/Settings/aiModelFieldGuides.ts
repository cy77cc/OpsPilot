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
