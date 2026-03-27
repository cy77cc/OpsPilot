export interface HelpDocument {
  id: string;
  title: string;
  content: string;
  category: string;
  tags: string[];
  difficulty: 'beginner' | 'intermediate' | 'advanced';
}

export const helpDocuments: HelpDocument[] = [
  {
    id: 'quick-start-login',
    title: '5 分钟快速上手平台',
    content:
      '1. 登录平台后先切换项目。\n2. 在「监控」页确认系统健康度与关键告警。\n3. 在「主机」页核查主机在线状态。\n4. 在「服务」页查看服务运行状态与版本。\n5. 遇到问题可打开右侧 AI 助手进行诊断。',
    category: '快速上手',
    tags: ['入门', '总览', '新手'],
    difficulty: 'beginner',
  },
  {
    id: 'host-onboard',
    title: '如何添加并纳管新主机？',
    content:
      '1. 进入「主机管理」->「主机纳管」。\n2. 填写主机 IP、端口与凭据，先执行连通性探测。\n3. 探测通过后提交纳管。\n4. 在主机列表确认状态为在线。',
    category: '主机管理',
    tags: ['主机', '纳管', 'SSH'],
    difficulty: 'beginner',
  },
  {
    id: 'host-troubleshooting',
    title: '主机探测失败怎么排查？',
    content:
      '优先检查 3 项：\n1. 网络连通（安全组/防火墙/端口）。\n2. 认证方式（密码或密钥是否匹配）。\n3. 账号权限（是否允许执行诊断命令）。\n可在主机详情页查看最近探测日志。',
    category: '主机管理',
    tags: ['故障排查', '探测', '认证'],
    difficulty: 'intermediate',
  },
  {
    id: 'monitor-alert',
    title: '如何处理高优先级告警？',
    content:
      '1. 在监控页按严重级别筛选 critical。\n2. 打开告警详情查看触发指标、时间窗与关联对象。\n3. 结合 AI 助手执行只读诊断（CPU/内存/磁盘/日志）。\n4. 处理后记录结果并关闭或恢复告警。',
    category: '监控告警',
    tags: ['告警', 'SRE', '值班'],
    difficulty: 'intermediate',
  },
  {
    id: 'service-release',
    title: '服务发布标准流程',
    content:
      '1. 在「服务管理」选择目标服务。\n2. 先执行发布预览，确认配置、镜像版本与变更范围。\n3. 生产环境变更需二次确认。\n4. 发布后观察健康指标与错误率，必要时执行回滚。',
    category: '服务管理',
    tags: ['发布', '回滚', '变更'],
    difficulty: 'intermediate',
  },
  {
    id: 'service-config',
    title: '配置中心发布与回滚',
    content:
      '1. 在配置中心编辑配置并保存草稿。\n2. 通过 Diff 对比确认变更内容。\n3. 发布到目标环境后观察应用状态。\n4. 异常时使用历史版本一键回滚。',
    category: '配置中心',
    tags: ['配置', 'diff', '回滚'],
    difficulty: 'intermediate',
  },
  {
    id: 'job-schedule',
    title: '如何创建定时任务？',
    content:
      '1. 进入「任务管理」->「创建任务」。\n2. 填写执行目标、命令和调度表达式。\n3. 先在测试环境验证，再启用生产调度。\n4. 在执行历史中检查输出与耗时。',
    category: '任务管理',
    tags: ['定时任务', '执行历史', '自动化'],
    difficulty: 'beginner',
  },
  {
    id: 'rbac-guide',
    title: '权限模型与最小权限实践',
    content:
      '1. 先创建角色，再绑定用户。\n2. 默认只授予读取权限，按需追加写入权限。\n3. 对生产环境操作建议使用审批与审计。\n4. 定期复核高权限账号。',
    category: '权限与安全',
    tags: ['RBAC', '权限', '安全'],
    difficulty: 'advanced',
  },
  {
    id: 'ai-assistant',
    title: '如何高效使用 AI 助手？',
    content:
      '推荐提问模板：\n- 诊断类："帮我排查 <对象> 的 <现象>，先给只读检查步骤"。\n- 变更类："请先给预览，再说明风险和回滚方案"。\n- 汇总类："基于当前告警给我一个 10 分钟内可执行计划"。',
    category: 'AI 助手',
    tags: ['AI', '提问模板', '效率'],
    difficulty: 'beginner',
  },
  {
    id: 'ai-safety',
    title: 'AI 执行安全与审批机制',
    content:
      '1. 默认只读工具可直接执行。\n2. 变更类操作必须审批确认。\n3. 审批超时或参数缺失会被拒绝执行。\n4. 所有执行均可在审计链路中追踪。',
    category: 'AI 助手',
    tags: ['审批', '安全', '审计'],
    difficulty: 'advanced',
  },
  {
    id: 'incident-playbook',
    title: '线上事故 15 分钟处置建议',
    content:
      '0-5 分钟：确认影响范围并冻结高风险变更。\n5-10 分钟：拉取关键指标、日志和事件，定位最可能故障点。\n10-15 分钟：执行回滚/降级/限流，恢复核心路径后再深挖根因。',
    category: '值班处置',
    tags: ['事故', '应急', 'playbook'],
    difficulty: 'advanced',
  },
];
