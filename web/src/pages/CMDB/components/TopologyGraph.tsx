import React, { useEffect, useRef } from 'react';
import type { Graph } from '@antv/g6';
import G6 from '@antv/g6';
import type { CMDBTopologyData } from '../../../api/modules/cmdb';
import { transformTopologyData } from '../utils/graph-helper';

interface TopologyGraphProps {
  data: CMDBTopologyData;
  onNodeSelect?: (nodeId: string | null) => void;
  onNodeDoubleClick?: (nodeId: string) => void;
  selectedCIID?: string | null;
}

const TopologyGraph: React.FC<TopologyGraphProps> = ({
  data,
  onNodeSelect,
  onNodeDoubleClick,
  selectedCIID,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<Graph | null>(null);

  useEffect(() => {
    if (!containerRef.current) {return;}

    const width = containerRef.current.scrollWidth;
    const height = containerRef.current.scrollHeight || 500;

    const graph = new G6.Graph({
      container: containerRef.current,
      width,
      height,
      layout: {
        type: 'dagre',
        rankdir: 'LR', // Left to Right
        nodesep: 30,
        ranksep: 50,
      },
      modes: {
        default: ['drag-canvas', 'zoom-canvas', 'drag-node', 'click-select'],
      },
      defaultNode: {
        type: 'rect',
        size: [120, 40],
        style: {
          radius: 5,
          stroke: '#5B8FF9',
          fill: '#C6E5FF',
          lineWidth: 2,
        },
        labelCfg: {
          style: {
            fill: '#000',
            fontSize: 12,
          },
        },
      },
      defaultEdge: {
        type: 'polyline',
        style: {
          stroke: '#e2e2e2',
          endArrow: true,
        },
      },
      nodeStateStyles: {
        selected: {
          stroke: '#1890ff',
          lineWidth: 3,
        },
        active: {
          opacity: 1,
        },
        inactive: {
          opacity: 0.2,
        },
      },
      edgeStateStyles: {
        active: {
          stroke: '#1890ff',
          lineWidth: 2,
        },
        inactive: {
          opacity: 0.1,
        },
      },
    });

    graph.on('node:click', (evt) => {
      const item = evt.item as any;
      if (item && item.getType() === 'node') {
        const id = item.get('id');
        onNodeSelect?.(id);
        
        // Highlight associated paths
        graph.setAutoPaint(false);
        graph.getNodes().forEach((node) => {
          graph.clearItemStates(node);
          graph.setItemState(node, 'inactive', true);
        });
        graph.getEdges().forEach((edge) => {
          graph.clearItemStates(edge);
          graph.setItemState(edge, 'inactive', true);
        });

        graph.setItemState(item, 'inactive', false);
        graph.setItemState(item, 'selected', true);
        
        const edges = item.getEdges();
        edges.forEach((edge: any) => {
          graph.setItemState(edge, 'inactive', false);
          graph.setItemState(edge, 'active', true);
          const source = edge.getSource();
          const target = edge.getTarget();
          graph.setItemState(source, 'inactive', false);
          graph.setItemState(target, 'inactive', false);
        });
        graph.paint();
        graph.setAutoPaint(true);
      }
    });

    graph.on('canvas:click', () => {
      onNodeSelect?.(null);
      graph.setAutoPaint(false);
      graph.getNodes().forEach((node) => {
        graph.clearItemStates(node);
      });
      graph.getEdges().forEach((edge) => {
        graph.clearItemStates(edge);
      });
      graph.paint();
      graph.setAutoPaint(true);
    });

    graph.on('node:dblclick', (evt) => {
      const item = evt.item;
      if (item) {
        onNodeDoubleClick?.(item.get('id'));
      }
    });

    graphRef.current = graph;

    const handleResize = () => {
      if (!graph || graph.get('destroyed')) {return;}
      if (!containerRef.current) {return;}
      graph.changeSize(containerRef.current.scrollWidth, containerRef.current.scrollHeight || 500);
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      graph.destroy();
    };
  }, []);

  useEffect(() => {
    if (graphRef.current && data) {
      const g6Data = transformTopologyData(data);
      graphRef.current.data(g6Data);
      graphRef.current.render();
      graphRef.current.fitView();
    }
  }, [data]);

  useEffect(() => {
    if (graphRef.current && selectedCIID) {
      const item = graphRef.current.findById(selectedCIID);
      if (item) {
        graphRef.current.focusItem(item, true);
        graphRef.current.setItemState(item, 'selected', true);
      }
    } else if (graphRef.current && !selectedCIID) {
        graphRef.current.getNodes().forEach(node => {
            graphRef.current?.setItemState(node, 'selected', false);
        });
    }
  }, [selectedCIID]);

  return (
    <div 
      ref={containerRef} 
      style={{ width: '100%', height: '100%', minHeight: '500px', background: '#f9f9f9' }} 
    />
  );
};

export default TopologyGraph;
