import React from 'react';
import { Select } from 'antd';
import { Api } from '../../api';
import type { Project } from '../../api/modules/projects';
import { useScope } from '../../app/scope/useScope';

const ProjectSwitcher: React.FC = () => {
  const [projects, setProjects] = React.useState<Project[]>([]);
  const { projectId, setProjectId } = useScope();

  React.useEffect(() => {
    const load = async () => {
      try {
        const res = await Api.projects.list();
        const list = res.data.list || [];
        setProjects(list);
        
        // 只有当当前没有选择项目，且列表不为空时，才设置默认项目
        if (!projectId && list.length > 0) {
          setProjectId(String(list[0].id));
        }
      } catch {
        setProjects([]);
      }
    };
    load();
    // 移除 [projectId, setProjectId] 依赖，仅在组件挂载时执行一次
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Select
      value={projectId}
      placeholder="选择项目组"
      style={{ width: 180 }}
      options={projects.map((p) => ({ value: p.id, label: p.name }))}
      onChange={(next) => {
        setProjectId(String(next));
      }}
    />
  );
};

export default ProjectSwitcher;
