import { useEffect, useState } from 'react';
import type { Project } from '../../lib/api';
import { getProjects } from '../../lib/api';
import type { Locale } from '../../lib/i18n';

interface Props {
  locale: Locale;
}

function monthlyTarget(project: Project): number {
  if (project.owner_want_monthly != null && project.owner_want_monthly > 0) {
    return project.owner_want_monthly;
  }
  if (project.costs) {
    return (
      project.costs.server_cost_monthly +
      project.costs.dev_cost_per_day * project.costs.dev_days_per_month +
      project.costs.other_cost_monthly
    );
  }
  return 0;
}

export default function ProjectList({ locale }: Props) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getProjects()
      .then(setProjects)
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <p>読み込み中...</p>;
  if (error) return <p style={{ color: 'var(--color-danger)' }}>{error}</p>;
  if (projects.length === 0) return <p>プロジェクトはまだありません。</p>;

  return (
    <div className="project-list" style={{ marginTop: '2rem' }}>
      {projects.map((project) => {
        const target = monthlyTarget(project);
        const achievementRate = target > 0 ? 0 : 0;
        return (
          <a key={project.id} href={`/projects/${project.id}`} className="card project-card">
            <div
              className="project-header"
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'flex-start',
                marginBottom: '0.5rem',
              }}
            >
              <h2 style={{ margin: 0 }}>{project.name}</h2>
              {target > 0 && (
                <span
                  className="achievement-badge"
                  data-level={achievementRate >= 80 ? 'ok' : achievementRate >= 50 ? 'warn' : 'danger'}
                >
                  {achievementRate}%
                </span>
              )}
            </div>
            <p style={{ margin: 0, color: 'var(--color-text-muted)', fontSize: '0.95rem' }}>
              {project.description || ''}
            </p>
            {target > 0 && (
              <p style={{ margin: '0.5rem 0 0', fontSize: '0.9rem' }}>
                月額目標: ¥{target.toLocaleString()}
              </p>
            )}
          </a>
        );
      })}
    </div>
  );
}
