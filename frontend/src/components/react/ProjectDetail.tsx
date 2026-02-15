import { useEffect, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import type { Project, ProjectUpdate } from '../../lib/api';
import { getProject, getProjectUpdates, PLATFORM_PROJECT_ID } from '../../lib/api';
import DonateForm from './DonateForm';
import ProjectChart from './charts/ProjectChart';
import { t, type Locale } from '../../lib/i18n';

interface Props {
  id: string;
  locale: Locale;
  backLabel: string;
  supportStatus: string;
  supportTitle: string;
  donateLabel: string;
  ownerLabel: string;
  recentSupportersLabel: string;
  anonymousLabel: string;
  donateFormPresets: number[];
  customAmountLabel: string;
  messageLabel: string;
  messagePlaceholder: string;
  thankYouTitle: string;
  donateLabelMonthly: string;
  oneTimeLabel: string;
  monthlyLabel: string;
  donationTypeLabel?: string;
  chartMinAmountLabel: string;
  chartTargetAmountLabel: string;
  chartActualAmountLabel: string;
  chartNoDataLabel: string;
  hostPageLink?: string;
  backHref?: string;
  hideHostPageLink?: boolean;
  tabSupportLabel: string;
  tabOverviewLabel: string;
  tabUpdatesLabel: string;
  updatesEmptyLabel: string;
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

function formatUpdateDate(iso: string, locale: Locale): string {
  const d = new Date(iso);
  const now = new Date();
  const diffDays = Math.floor((now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24));
  const loc = locale === 'ja' ? 'ja-JP' : 'en-US';
  if (diffDays === 0) return d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' });
  if (diffDays === 1) return locale === 'ja' ? '昨日' : 'Yesterday';
  if (diffDays < 7) return locale === 'ja' ? `${diffDays}日前` : `${diffDays} days ago`;
  return d.toLocaleDateString(loc, { year: 'numeric', month: 'short', day: 'numeric' });
}

type TabId = 'support' | 'overview' | 'updates';

export default function ProjectDetail({
  id,
  locale,
  backLabel,
  supportStatus,
  supportTitle,
  donateLabel,
  ownerLabel,
  recentSupportersLabel,
  anonymousLabel,
  donateFormPresets,
  customAmountLabel,
  messageLabel,
  messagePlaceholder,
  thankYouTitle,
  donateLabelMonthly,
  oneTimeLabel,
  monthlyLabel,
  donationTypeLabel,
  chartMinAmountLabel,
  chartTargetAmountLabel,
  chartActualAmountLabel,
  chartNoDataLabel,
  hostPageLink,
  backHref,
  hideHostPageLink,
  tabSupportLabel,
  tabOverviewLabel,
  tabUpdatesLabel,
  updatesEmptyLabel,
}: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [updates, setUpdates] = useState<ProjectUpdate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<TabId>('support');

  useEffect(() => {
    getProject(id)
      .then(setProject)
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (id) {
      getProjectUpdates(id).then(setUpdates).catch(() => setUpdates([]));
    }
  }, [id]);

  if (loading) return <p>{t(locale, 'projects.loading')}</p>;
  if (error) return <p style={{ color: 'var(--color-danger)' }}>{error}</p>;
  if (!project) return null;

  const target = monthlyTarget(project);
  const currentMonthly = project.current_monthly_donations ?? 0;
  const achievementRate = target > 0 ? Math.round((currentMonthly / target) * 100) : 0;

  const basePath = locale === 'en' ? '/en' : '';
  const backUrl = backHref ?? `${basePath}/projects`;

  const overview = project.overview ?? project.description ?? '';

  return (
    <div className="project-detail">
      <a href={backUrl} style={{ color: 'var(--color-primary)', textDecoration: 'none', fontSize: '0.9rem' }}>
        ← {backLabel}
      </a>

      {/* ヒーロー画像 */}
      <div
        className="project-hero"
        style={{
          marginTop: '1rem',
          marginBottom: '1.5rem',
          borderRadius: '12px',
          overflow: 'hidden',
          backgroundColor: 'var(--color-bg-subtle)',
          aspectRatio: '800 / 400',
          maxHeight: '400px',
        }}
      >
        {project.image_url ? (
          <img
            src={project.image_url}
            alt=""
            style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
          />
        ) : (
          <div
            style={{
              width: '100%',
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '4rem',
              color: 'var(--color-primary-muted)',
            }}
          >
            📦
          </div>
        )}
      </div>

      <h1 style={{ marginTop: 0 }}>
        {project.name}
        {project.id === PLATFORM_PROJECT_ID && hostPageLink && !hideHostPageLink && (
          <a href={`${basePath}/host`} style={{ marginLeft: '0.5rem', fontSize: '0.75rem', fontWeight: 500, color: 'var(--color-primary)' }}>
            ({hostPageLink})
          </a>
        )}
      </h1>
      {project.owner_name && (
        <p style={{ marginTop: '0.25rem', color: 'var(--color-text-muted)', fontSize: '0.95rem' }}>
          {ownerLabel}: {project.owner_name}
        </p>
      )}
      <p style={{ marginTop: '0.5rem', color: 'var(--color-text-muted)' }}>{project.description || ''}</p>

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

      {/* タブ */}
      <div
        className="project-tabs"
        style={{
          marginTop: '2rem',
          borderBottom: '2px solid var(--color-border)',
          display: 'flex',
          gap: '0.5rem',
        }}
      >
        {(['support', 'overview', 'updates'] as TabId[]).map((tabId) => (
          <button
            key={tabId}
            type="button"
            onClick={() => setActiveTab(tabId)}
            style={{
              padding: '0.75rem 1.25rem',
              border: 'none',
              background: 'none',
              cursor: 'pointer',
              fontSize: '1rem',
              fontWeight: activeTab === tabId ? 600 : 500,
              color: activeTab === tabId ? 'var(--color-primary)' : 'var(--color-text-muted)',
              borderBottom: activeTab === tabId ? '2px solid var(--color-primary)' : '2px solid transparent',
              marginBottom: '-2px',
            }}
          >
            {tabId === 'support' && tabSupportLabel}
            {tabId === 'overview' && tabOverviewLabel}
            {tabId === 'updates' && tabUpdatesLabel}
          </button>
        ))}
      </div>

      {/* タブコンテンツ */}
      <div className="project-tab-content" style={{ marginTop: '1.5rem' }}>
        {activeTab === 'support' && (
          <>
            <div className="card" style={{ marginBottom: '1.5rem' }}>
              <h2 style={{ marginTop: 0 }}>{supportStatus}</h2>
              <div className="achievement-bar" style={{ margin: '1rem 0' }}>
                <div
                  className="achievement-fill"
                  style={{ width: `${Math.min(achievementRate, 100)}%` }}
                />
              </div>
              <p>
                {t(locale, 'projects.supportStatusDetail', {
                  target: target.toLocaleString(),
                  current: currentMonthly.toLocaleString(),
                  rate: String(achievementRate),
                })}
              </p>
              <ProjectChart
                projectId={project.id}
                minAmountLabel={chartMinAmountLabel}
                targetAmountLabel={chartTargetAmountLabel}
                actualAmountLabel={chartActualAmountLabel}
                noDataLabel={chartNoDataLabel}
              />
            </div>

            {project.recent_supporters && project.recent_supporters.length > 0 && (
              <div className="card" style={{ marginBottom: '1.5rem' }}>
                <h2 style={{ fontSize: '1.1rem', marginTop: 0 }}>{recentSupportersLabel}</h2>
                <ul style={{ margin: '0.5rem 0 0', paddingLeft: '1.25rem', fontSize: '0.95rem' }}>
                  {project.recent_supporters.slice(0, 5).map((s, i) => (
                    <li key={i}>
                      {s.name ?? anonymousLabel}: ¥{s.amount.toLocaleString()}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <div className="card accent-line">
              <h2 style={{ marginTop: 0 }}>{supportTitle}</h2>
              <DonateForm
                locale={locale}
                projectName={project.name}
                donateLabel={donateLabel}
                amountPresets={donateFormPresets}
                customAmountLabel={customAmountLabel}
                messageLabel={messageLabel}
                messagePlaceholder={messagePlaceholder}
                submitLabel={donateLabel}
                submitLabelMonthly={donateLabelMonthly}
                thankYouTitle={thankYouTitle}
                thankYouMessageKey="projects.thankYouMessage"
                thankYouMessageMonthlyKey="projects.thankYouMessageMonthly"
                oneTimeLabel={oneTimeLabel}
                monthlyLabel={monthlyLabel}
                donationTypeLabel={donationTypeLabel}
              />
            </div>
          </>
        )}

        {activeTab === 'overview' && (
          <div className="card project-overview-markdown" style={{ padding: '1.5rem' }}>
            <div style={{ maxWidth: '65ch' }}>
              <ReactMarkdown
                components={{
                  h1: ({ children }) => <h1 style={{ marginTop: '1.5rem', marginBottom: '0.5rem', fontSize: '1.4rem' }}>{children}</h1>,
                  h2: ({ children }) => <h2 style={{ marginTop: '1.5rem', marginBottom: '0.5rem', fontSize: '1.2rem' }}>{children}</h2>,
                  h3: ({ children }) => <h3 style={{ marginTop: '1.25rem', marginBottom: '0.5rem', fontSize: '1.1rem' }}>{children}</h3>,
                  p: ({ children }) => <p style={{ margin: '0.5rem 0', lineHeight: 1.7 }}>{children}</p>,
                  ul: ({ children }) => <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem' }}>{children}</ul>,
                  ol: ({ children }) => <ol style={{ margin: '0.5rem 0', paddingLeft: '1.5rem' }}>{children}</ol>,
                  li: ({ children }) => <li style={{ margin: '0.25rem 0' }}>{children}</li>,
                  a: ({ href, children }) => <a href={href} style={{ color: 'var(--color-primary)', textDecoration: 'underline' }} target="_blank" rel="noopener noreferrer">{children}</a>,
                  strong: ({ children }) => <strong style={{ fontWeight: 600 }}>{children}</strong>,
                  pre: ({ children }) => <pre style={{ backgroundColor: 'var(--color-bg-subtle)', padding: '1rem', borderRadius: '8px', overflow: 'auto', fontSize: '0.9em' }}>{children}</pre>,
                  code: ({ children }) => <code style={{ backgroundColor: 'var(--color-bg-subtle)', padding: '0.1rem 0.3rem', borderRadius: '4px', fontSize: '0.9em' }}>{children}</code>,
                }}
              >
                {overview}
              </ReactMarkdown>
            </div>
          </div>
        )}

        {activeTab === 'updates' && (
          <div className="card" style={{ padding: '1.5rem' }}>
            {updates.length === 0 ? (
              <p style={{ color: 'var(--color-text-muted)' }}>{updatesEmptyLabel}</p>
            ) : (
              <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                {updates.map((u) => (
                  <li
                    key={u.id}
                    style={{
                      padding: '1rem 0',
                      borderBottom: '1px solid var(--color-border-light)',
                    }}
                  >
                    <div style={{ fontSize: '0.85rem', color: 'var(--color-text-muted)', marginBottom: '0.25rem' }}>
                      {u.author_name && <span>{u.author_name}</span>}
                      <span style={{ marginLeft: '0.5rem' }}>{formatUpdateDate(u.created_at, locale)}</span>
                    </div>
                    {u.title && <h3 style={{ margin: '0.25rem 0', fontSize: '1rem' }}>{u.title}</h3>}
                    <p style={{ margin: 0, whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>{u.body}</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
