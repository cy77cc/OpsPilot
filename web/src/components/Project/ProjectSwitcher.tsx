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
        if (!projectId && list.length > 0) {
          const first = list[0].id;
          setProjectId(first);
        }
      } catch {
        setProjects([]);
      }
    };
    load();
  }, [projectId, setProjectId]);

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
