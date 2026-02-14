import { useEffect, useState } from 'react';
import type { Project } from '../../lib/api';
import { getProject } from '../../lib/api';
import type { Locale } from '../../lib/i18n';
import ProjectForm from './ProjectForm';

interface Props {
  locale: Locale;
  projectId: string;
  redirectPath: string;
}

export default function ProjectFormWithLoader({ locale, projectId, redirectPath }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getProject(projectId)
      .then(setProject)
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setLoading(false));
  }, [projectId]);

  if (loading) return <p>読み込み中...</p>;
  if (error) return <p style={{ color: 'var(--color-danger)' }}>{error}</p>;
  if (!project) return null;

  return <ProjectForm locale={locale} project={project} redirectPath={redirectPath} />;
}
