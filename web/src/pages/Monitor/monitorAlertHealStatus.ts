export interface HealStatusDisplay {
  processing: string;
  healing: string;
  processingColor: string;
  healingColor: string;
}

export function normalizeHealStatus(status: string): HealStatusDisplay {
  switch (status) {
    case 'waiting_approval':
      return { processing: '待人工', healing: '转人工审批', processingColor: 'orange', healingColor: 'volcano' };
    case 'auto_fixing':
      return { processing: '处理中', healing: '自动修复中', processingColor: 'blue', healingColor: 'geekblue' };
    case 'succeeded':
      return { processing: '已处理', healing: 'AI自愈成功', processingColor: 'green', healingColor: 'cyan' };
    case 'failed_manual':
      return { processing: '待人工', healing: 'AI修复失败', processingColor: 'orange', healingColor: 'red' };
    case 'no_action':
      return { processing: '已处理', healing: 'AI判定无需处理', processingColor: 'green', healingColor: 'gold' };
    case 'canceled_resolved':
      return { processing: '已处理', healing: '告警恢复已取消', processingColor: 'green', healingColor: 'default' };
    default:
      return { processing: '待处理', healing: '待分析', processingColor: 'default', healingColor: 'default' };
  }
}
