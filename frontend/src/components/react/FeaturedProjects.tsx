import { useEffect, useState } from 'react';
import type { Project } from '../../lib/api';
import { getNewProjects, getHotProjects } from '../../lib/api';
import type { Locale } from '../../lib/i18n';

interface Props {
  locale: Locale;
  newTitle: string;
  hotTitle: string;
  viewAll: string;
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

function achievementRate(project: Project): number {
  const target = monthlyTarget(project);
  const current = project.current_monthly_donations ?? 0;
  return target > 0 ? Math.round((current / target) * 100) : 0;
}

function ProjectCard({ project }: { project: Project }) {
  const target = monthlyTarget(project);
  const rate = achievementRate(project);
  return (
    <a href={`/projects/${project.id}`} className="card project-card" style={{ display: 'block', textDecoration: 'none', color: 'inherit' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.5rem' }}>
        <h3 style={{ margin: 0, fontSize: '1.1rem' }}>{project.name}</h3>
        {target > 0 && (
          <span
            className="achievement-badge"
            data-level={rate >= 80 ? 'ok' : rate >= 50 ? 'warn' : 'danger'}
          >
            {rate}%
          </span>
        )}
      </div>
      <p style={{ margin: 0, color: 'var(--color-text-muted)', fontSize: '0.9rem', lineHeight: 1.4 }}>
        {project.description || ''}
      </p>
      {target > 0 && (
        <p style={{ margin: '0.5rem 0 0', fontSize: '0.85rem' }}>
          月額目標: ¥{target.toLocaleString()}
        </p>
      )}
    </a>
  );
}

export default function FeaturedProjects({ locale, newTitle, hotTitle, viewAll }: Props) {
  const [newProjects, setNewProjects] = useState<Project[]>([]);
  const [hotProjects, setHotProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([getNewProjects(5), getHotProjects(5)])
      .then(([newP, hotP]) => {
        setNewProjects(newP);
        setHotProjects(hotP);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div style={{ marginTop: '2rem' }}>
        <p style={{ color: 'var(--color-text-muted)' }}>読み込み中...</p>
      </div>
    );
  }

  const hasNew = newProjects.length > 0;
  const hasHot = hotProjects.length > 0;
  if (!hasNew && !hasHot) return null;

  return (
    <div className="featured-projects" style={{ marginTop: '2rem' }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '2rem' }}>
        {hasNew && (
          <section className="card">
            <h2 style={{ marginTop: 0, marginBottom: '1rem', fontSize: '1.2rem' }}>{newTitle}</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              {newProjects.map((p) => (
                <ProjectCard key={p.id} project={p} />
              ))}
            </div>
            <a href="/projects" style={{ display: 'inline-block', marginTop: '1rem', fontSize: '0.9rem', color: 'var(--color-primary)' }}>
              {viewAll} →
            </a>
          </section>
        )}
        {hasHot && (
          <section className="card">
            <h2 style={{ marginTop: 0, marginBottom: '1rem', fontSize: '1.2rem' }}>{hotTitle}</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              {hotProjects.map((p) => (
                <ProjectCard key={p.id} project={p} />
              ))}
            </div>
            <a href="/projects" style={{ display: 'inline-block', marginTop: '1rem', fontSize: '0.9rem', color: 'var(--color-primary)' }}>
              {viewAll} →
            </a>
          </section>
        )}
      </div>
    </div>
  );
}
