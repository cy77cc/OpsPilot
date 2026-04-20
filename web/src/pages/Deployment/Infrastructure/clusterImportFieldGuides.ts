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
