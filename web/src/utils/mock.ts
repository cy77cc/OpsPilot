import { faker } from '@faker-js/faker';

// 类型定义
export interface Host {
  id: string;
  hostname: string;
  ip: string;
  cpuUsage: number;
  memoryUsage: number;
  status: 'RUNNING' | 'WARNING' | 'ERROR' | 'OFFLINE';
  lastUpdated: Date;
}

export interface Cluster {
  id: string;
  name: string;
  nodeCount: number;
  cpuUsed: number;
  cpuTotal: number;
  memoryUsed: number;
  memoryTotal: number;
}

export interface ConfigItem {
  id: string;
  name: string;
  content: string;  // YAML 格式
  version: string;  // 版本号
  commitHash: string;
  updatedAt: Date;
}

export interface Task {
  id: string;
  name: string;
  executeTime: Date;
  status: 'SUCCESS' | 'FAILURE';
  errorLog?: string;  // 失败时的错误日志
}

// 生成主机数据
const generateHosts = (): Host[] => {
  const hosts: Host[] = [];
  
  // 生成 17 台正常运行的主机
  for (let i = 0; i < 17; i++) {
    hosts.push({
      id: faker.string.uuid(),
      hostname: `host-${faker.string.alphanumeric(5)}`,
      ip: faker.internet.ipv4(),
      cpuUsage: faker.number.int({ min: 10, max: 70 }),
      memoryUsage: faker.number.int({ min: 20, max: 60 }),
      status: 'RUNNING',
      lastUpdated: new Date()
    });
  }
  
  // 生成 2 台警告状态的主机（CPU 使用率 > 85%）
  for (let i = 0; i < 2; i++) {
    hosts.push({
      id: faker.string.uuid(),
      hostname: `host-${faker.string.alphanumeric(5)}`,
      ip: faker.internet.ipv4(),
      cpuUsage: faker.number.int({ min: 86, max: 95 }),
      memoryUsage: faker.number.int({ min: 70, max: 85 }),
      status: 'WARNING',
      lastUpdated: new Date()
    });
  }
  
  // 生成 1 台离线状态的主机
  hosts.push({
    id: faker.string.uuid(),
    hostname: `host-${faker.string.alphanumeric(5)}`,
    ip: faker.internet.ipv4(),
    cpuUsage: 0,
    memoryUsage: 0,
    status: 'OFFLINE',
    lastUpdated: new Date()
  });
  
  return hosts;
};

// 生成集群数据
const generateClusters = (): Cluster[] => {
  return [
    {
      id: faker.string.uuid(),
      name: 'Production Cluster',
      nodeCount: 8,
      cpuUsed: 12.5,
      cpuTotal: 32,
      memoryUsed: 45.2,
      memoryTotal: 128
    },
    {
      id: faker.string.uuid(),
      name: 'Staging Cluster',
      nodeCount: 4,
      cpuUsed: 5.2,
      cpuTotal: 16,
      memoryUsed: 18.5,
      memoryTotal: 64
    },
    {
      id: faker.string.uuid(),
      name: 'Development Cluster',
      nodeCount: 2,
      cpuUsed: 1.8,
      cpuTotal: 8,
      memoryUsed: 6.3,
      memoryTotal: 32
    }
  ];
};

// 生成配置中心数据
const generateConfigItems = (): ConfigItem[] => {
  return [
    {
      id: faker.string.uuid(),
      name: 'application.yml',
      content: `server:\n  port: 8080\n\nspring:\n  application:\n    name: devops-service\n  profiles:\n    active: prod`,
      version: 'v1.0.1',
      commitHash: faker.git.commitSha(),
      updatedAt: new Date()
    },
    {
      id: faker.string.uuid(),
      name: 'database.yml',
      content: `database:\n  url: jdbc:mysql://localhost:3306/devops\n  username: admin\n  password: password`,
      version: 'v1.0.0',
      commitHash: faker.git.commitSha(),
      updatedAt: new Date()
    }
  ];
};

// 生成任务调度数据
const generateTasks = (): Task[] => {
  const tasks: Task[] = [];
  
  // 生成 7 个成功任务和 3 个失败任务
  for (let i = 0; i < 7; i++) {
    tasks.push({
      id: faker.string.uuid(),
      name: `Task ${faker.string.alphanumeric(5)}`,
      executeTime: faker.date.recent(),
      status: 'SUCCESS'
    });
  }
  
  for (let i = 0; i < 3; i++) {
    tasks.push({
      id: faker.string.uuid(),
      name: `Task ${faker.string.alphanumeric(5)}`,
      executeTime: faker.date.recent(),
      status: 'FAILURE',
      errorLog: `Error: ${faker.lorem.paragraph(2)}. This is a mock error log with at least 50 characters.`
    });
  }
  
  return tasks;
};

// 模拟数据生成函数
export const useMockData = () => {
  return {
    generateHosts,
    generateClusters,
    generateConfigItems,
    generateTasks
  };
};