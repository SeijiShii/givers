import { useEffect, useState } from 'react';
import type { Project } from '../../lib/api';
import { getProject } from '../../lib/api';
import type { Locale } from '../../lib/i18n';

interface Props {
  id: string;
  locale: Locale;
  backLabel: string;
  supportStatus: string;
  supportStatusDetail: (args: { target: string; current: string; rate: string }) => string;
  supportTitle: string;
  phaseNoteDonate: string;
  donateLabel: string;
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

export default function ProjectDetail({
  id,
  backLabel,
  supportStatus,
  supportStatusDetail,
  supportTitle,
  phaseNoteDonate,
  donateLabel,
}: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getProject(id)
      .then(setProject)
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <p>読み込み中...</p>;
  if (error) return <p style={{ color: 'var(--color-danger)' }}>{error}</p>;
  if (!project) return null;

  const target = monthlyTarget(project);
  const currentMonthly = 0;
  const achievementRate = target > 0 ? Math.round((currentMonthly / target) * 100) : 0;

  return (
    <div className="project-detail">
      <a href="/projects" style={{ color: 'var(--color-primary)', textDecoration: 'none', fontSize: '0.9rem' }}>
        ← {backLabel}
      </a>

      <h1>{project.name}</h1>
      <p>{project.description || ''}</p>

      {(project.owner_want_monthly != null && project.owner_want_monthly > 0) && (
        <p style={{ marginTop: '1rem', fontWeight: 600 }}>
          最低希望額: 月額 ¥{project.owner_want_monthly.toLocaleString()}
        </p>
      )}
      {project.costs && (project.costs.server_cost_monthly > 0 || project.costs.dev_cost_per_day > 0 || project.costs.other_cost_monthly > 0) && (
        <p style={{ marginTop: '0.5rem', color: 'var(--color-text-muted)' }}>
          必要額（コスト内訳）: 月額 ¥{(
            project.costs.server_cost_monthly +
            project.costs.dev_cost_per_day * project.costs.dev_days_per_month +
            project.costs.other_cost_monthly
          ).toLocaleString()}
        </p>
      )}

      <div className="card" style={{ marginTop: '1.5rem' }}>
        <h2>{supportStatus}</h2>
        <div className="achievement-bar" style={{ margin: '1rem 0' }}>
          <div
            className="achievement-fill"
            style={{ width: `${Math.min(achievementRate, 100)}%` }}
          />
        </div>
        <p>
          {supportStatusDetail({
            target: target.toLocaleString(),
            current: currentMonthly.toLocaleString(),
            rate: String(achievementRate),
          })}
        </p>
      </div>

      <div className="card accent-line" style={{ marginTop: '1.5rem' }}>
        <h2>{supportTitle}</h2>
        <p>{phaseNoteDonate}</p>
        <button type="button" className="btn btn-accent" style={{ marginTop: '1rem' }}>
          {donateLabel}
        </button>
      </div>
    </div>
  );
}
