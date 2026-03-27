import React from 'react';
import { Select } from 'antd';
import { Api } from '../../api';
import type { Project } from '../../api/modules/projects';

const ProjectSwitcher: React.FC = () => {
  const [projects, setProjects] = React.useState<Project[]>([]);
  const [value, setValue] = React.useState<string | undefined>(localStorage.getItem('projectId') || undefined);

  React.useEffect(() => {
    const load = async () => {
      try {
        const res = await Api.projects.list();
        const list = res.data.list || [];
        setProjects(list);
        if (!value && list.length > 0) {
          const first = list[0].id;
          setValue(first);
          localStorage.setItem('projectId', first);
        }
      } catch {
        setProjects([]);
      }
    };
    load();
  }, []);

  return (
    <Select
      value={value}
      placeholder="选择项目组"
      style={{ width: 180 }}
      options={projects.map((p) => ({ value: p.id, label: p.name }))}
      onChange={(next) => {
        setValue(next);
        localStorage.setItem('projectId', String(next));
        window.dispatchEvent(new CustomEvent('project:changed', { detail: { projectId: next } }));
      }}
    />
  );
};

export default ProjectSwitcher;
