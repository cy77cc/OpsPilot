import { useRequest } from 'ahooks';
import { cmdbApi } from '../api/modules/cmdb';

/**
 * Hook for fetching CMDB tree data
 * @param parentId Parent node ID
 * @param viewType View type
 */
export const useCMDBTree = (parentId?: number, viewType?: string) => {
  return useRequest(
    () => cmdbApi.getTree({ parentId, viewType }),
    {
      refreshDeps: [parentId, viewType],
    }
  );
};

/**
 * Hook for fetching CMDB topology subgraph
 * @param rootId Root node ID
 * @param depth Traversal depth
 */
export const useCMDBTopology = (rootId: number, depth?: number) => {
  return useRequest(
    () => cmdbApi.getTopologySubgraph({ rootId, depth }),
    {
      refreshDeps: [rootId, depth],
      manual: !rootId,
    }
  );
};

/**
 * Hook for fetching CMDB asset details
 * @param id Asset ID
 */
export const useCMDBAsset = (id?: string) => {
  return useRequest(
    () => id ? cmdbApi.getAsset(id) : Promise.reject('No asset ID provided'),
    {
      refreshDeps: [id],
      manual: !id,
    }
  );
};

/**
 * Hook for fetching CMDB asset audit logs
 * @param id Asset ID
 */
export const useCMDBAudits = (id?: string) => {
  return useRequest(
    () => id ? cmdbApi.listAudits(id) : Promise.reject('No asset ID provided'),
    {
      refreshDeps: [id],
      manual: !id,
    }
  );
};
