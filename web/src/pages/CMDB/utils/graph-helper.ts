import type { CMDBTopologyData } from '../../../api/modules/cmdb';

export interface G6Node {
  id: string;
  label: string;
  type?: string;
  size?: number | number[];
  style?: Record<string, any>;
  labelCfg?: Record<string, any>;
  icon?: Record<string, any>;
  stateStyles?: Record<string, any>;
  [key: string]: any;
}

export interface G6Edge {
  id: string;
  source: string;
  target: string;
  label?: string;
  type?: string;
  style?: Record<string, any>;
  labelCfg?: Record<string, any>;
  stateStyles?: Record<string, any>;
  [key: string]: any;
}

export interface G6Data {
  nodes: G6Node[];
  edges: G6Edge[];
}

const CI_TYPE_CONFIG: Record<string, { color: string; icon: string }> = {
  host: { color: '#1890ff', icon: 'desktop' },
  service: { color: '#52c41a', icon: 'api' },
  cluster: { color: '#722ed1', icon: 'cluster' },
  database: { color: '#fa8c16', icon: 'database' },
  app: { color: '#2f54eb', icon: 'appstore' },
  network: { color: '#eb2f96', icon: 'global' },
  default: { color: '#8c8c8c', icon: 'question' },
};

export const transformTopologyData = (data: CMDBTopologyData): G6Data => {
  const nodes: G6Node[] = (data.nodes || []).map((node) => {
    const config = CI_TYPE_CONFIG[node.type] || CI_TYPE_CONFIG.default;
    return {
      id: node.id,
      label: node.label || node.name || node.id,
      type: 'rect', // G6 built-in rect
      size: [120, 40],
      style: {
        fill: '#fff',
        stroke: config.color,
        lineWidth: 2,
        radius: 4,
      },
      labelCfg: {
        style: {
          fill: '#000',
          fontSize: 12,
        },
      },
      // G6 icon configuration if using custom node or specific built-in that supports it
      // For simplicity with built-in rect, we use stroke color to distinguish types
      ci_type: node.type,
      status: node.status,
    };
  });

  const edges: G6Edge[] = (data.edges || []).map((edge) => {
    const isDash = edge.type === 'dependency' || edge.relation_type === 'depends_on';
    return {
      id: edge.id || `${edge.source}-${edge.target}`,
      source: edge.source,
      target: edge.target,
      label: edge.label || edge.relation_type,
      type: 'polyline',
      style: {
        stroke: '#e2e2e2',
        lineDash: isDash ? [4, 4] : undefined,
        endArrow: {
          path: 'M 0,0 L 8,4 L 8,-4 Z',
          fill: '#e2e2e2',
        },
      },
      labelCfg: {
        autoRotate: true,
        style: {
          fontSize: 10,
          background: {
            fill: '#fff',
            padding: [2, 4],
            radius: 2,
          },
        },
      },
    };
  });

  return { nodes, edges };
};

export const getNodeStyleByType = (type: string) => {
  return CI_TYPE_CONFIG[type] || CI_TYPE_CONFIG.default;
};
