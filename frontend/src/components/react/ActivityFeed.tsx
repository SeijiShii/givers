import { useEffect, useState } from 'react';
import { getActivityFeed, type ActivityItem } from '../../lib/api';
import { t, type Locale } from '../../lib/i18n';

interface Props {
  locale: Locale;
  title: string;
  projectCreated: string;
  projectUpdated: string;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffM = Math.floor(diffMs / 60000);
  const diffH = Math.floor(diffMs / 3600000);
  const diffD = Math.floor(diffMs / 86400000);
  if (diffM < 60) return `${diffM}分前`;
  if (diffH < 24) return `${diffH}時間前`;
  if (diffD < 7) return `${diffD}日前`;
  return d.toLocaleDateString('ja-JP', { month: 'short', day: 'numeric' });
}

function renderWithProjectLink(
  template: string,
  projectName: string,
  projectId: string,
  replacements: Record<string, string>
): React.ReactNode {
  let text = template;
  for (const [key, val] of Object.entries(replacements)) {
    text = text.replace(`{${key}}`, val);
  }
  const parts = text.split('{project}');
  if (parts.length === 1) return text;
  const result: React.ReactNode[] = [];
  for (let i = 0; i < parts.length; i++) {
    result.push(parts[i]);
    if (i < parts.length - 1) {
      result.push(
        <a key={i} href={`/projects/${projectId}`} className="activity-project">
          {projectName}
        </a>
      );
    }
  }
  return <>{result}</>;
}

function ActivityLine({
  item,
  locale,
  projectCreated,
  projectUpdated,
}: {
  item: ActivityItem;
  locale: Locale;
  projectCreated: string;
  projectUpdated: string;
}) {
  const amountStr = item.amount != null ? `¥${item.amount.toLocaleString()}` : '';

  switch (item.type) {
    case 'project_created':
      return (
        <span className="activity-item">
          {renderWithProjectLink(
            projectCreated,
            item.project_name,
            item.project_id,
            { actor: item.actor_name ?? '' }
          )}
        </span>
      );
    case 'project_updated':
      return (
        <span className="activity-item">
          {renderWithProjectLink(
            projectUpdated,
            item.project_name,
            item.project_id,
            { actor: item.actor_name ?? '' }
          )}
        </span>
      );
    case 'donation':
      return (
        <span className="activity-item">
          {item.actor_name
            ? renderWithProjectLink(
                t(locale, 'feed.donationBy', { actor: item.actor_name, amount: amountStr }),
                item.project_name,
                item.project_id,
                {}
              )
            : renderWithProjectLink(
                t(locale, 'feed.donationAnonymous', { amount: amountStr }),
                item.project_name,
                item.project_id,
                {}
              )}
        </span>
      );
    case 'milestone':
      return (
        <span className="activity-item">
          {renderWithProjectLink(
            t(locale, 'feed.milestoneReached', { rate: String(item.rate ?? 0) }),
            item.project_name,
            item.project_id,
            {}
          )}
        </span>
      );
    default:
      return null;
  }
}

export default function ActivityFeed({
  locale,
  title,
  projectCreated,
  projectUpdated,
}: Props) {
  const [items, setItems] = useState<ActivityItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getActivityFeed(10)
      .then(setItems)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <section className="card activity-feed" style={{ marginTop: '2rem' }}>
        <h2 style={{ marginTop: 0, marginBottom: '1rem', fontSize: '1.2rem' }}>{title}</h2>
        <p style={{ color: 'var(--color-text-muted)' }}>読み込み中...</p>
      </section>
    );
  }

  if (items.length === 0) return null;

  return (
    <section className="card activity-feed" style={{ marginTop: '2rem' }}>
      <h2 style={{ marginTop: 0, marginBottom: '1rem', fontSize: '1.2rem' }}>{title}</h2>
      <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
        {items.map((item) => (
          <li
            key={item.id}
            style={{
              padding: '0.5rem 0',
              borderBottom: '1px solid var(--color-border-light)',
              fontSize: '0.95rem',
            }}
          >
            <span style={{ color: 'var(--color-text-muted)', marginRight: '0.5rem', fontSize: '0.85rem' }}>
              {formatTime(item.created_at)}
            </span>
            <ActivityLine
              item={item}
              locale={locale}
              projectCreated={projectCreated}
              projectUpdated={projectUpdated}
            />
          </li>
        ))}
      </ul>
    </section>
  );
}
