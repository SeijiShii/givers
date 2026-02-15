import { useEffect, useState } from 'react';
import { getMe, getAdminUsers, type User, type AdminUser } from '../../lib/api';
import type { Locale } from '../../lib/i18n';
import { t } from '../../lib/i18n';

interface Props {
  locale: Locale;
  title: string;
  forbiddenMessage: string;
  statusActive: string;
  statusSuspended: string;
  suspendLabel: string;
  unsuspendLabel: string;
  projectCountLabel: string;
}

export default function AdminUserList({
  locale,
  title,
  forbiddenMessage,
  statusActive,
  statusSuspended,
  suspendLabel,
  unsuspendLabel,
  projectCountLabel,
}: Props) {
  const [me, setMe] = useState<User | null>(null);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMe()
      .then(setMe)
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (me?.role === 'host') {
      getAdminUsers()
        .then(setUsers)
        .catch(() => setUsers([]));
    }
  }, [me?.role]);

  if (loading) {
    return <p>{t(locale, 'projects.loading')}</p>;
  }

  if (!me || me.role !== 'host') {
    return (
      <div className="card" style={{ padding: '1.5rem' }}>
        <p style={{ color: 'var(--color-text-muted)', margin: 0 }}>{forbiddenMessage}</p>
      </div>
    );
  }

  return (
    <div className="admin-users">
      <h1>{title}</h1>
      <div className="card" style={{ marginTop: '1rem', overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid var(--color-border)' }}>
              <th style={{ padding: '0.75rem', textAlign: 'left' }}>{locale === 'ja' ? '名前' : 'Name'}</th>
              <th style={{ padding: '0.75rem', textAlign: 'left' }}>{locale === 'ja' ? 'メール' : 'Email'}</th>
              <th style={{ padding: '0.75rem', textAlign: 'left' }}>{locale === 'ja' ? 'ステータス' : 'Status'}</th>
              <th style={{ padding: '0.75rem', textAlign: 'right' }}>{projectCountLabel}</th>
              <th style={{ padding: '0.75rem' }}>{locale === 'ja' ? '操作' : 'Actions'}</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id} style={{ borderBottom: '1px solid var(--color-border-light)' }}>
                <td style={{ padding: '0.75rem' }}>{u.name}</td>
                <td style={{ padding: '0.75rem', fontSize: '0.9rem' }}>{u.email}</td>
                <td style={{ padding: '0.75rem' }}>
                  <span
                    style={{
                      padding: '0.2rem 0.5rem',
                      borderRadius: '4px',
                      fontSize: '0.85rem',
                      backgroundColor: u.status === 'active' ? 'var(--color-primary-muted)' : 'var(--color-danger)',
                      color: u.status === 'active' ? 'var(--color-text)' : 'white',
                    }}
                  >
                    {u.status === 'active' ? statusActive : statusSuspended}
                  </span>
                </td>
                <td style={{ padding: '0.75rem', textAlign: 'right' }}>{u.project_count ?? 0}</td>
                <td style={{ padding: '0.75rem' }}>
                  {u.status === 'active' ? (
                    <button type="button" className="btn" style={{ fontSize: '0.85rem', padding: '0.25rem 0.5rem' }}>
                      {suspendLabel}
                    </button>
                  ) : (
                    <button type="button" className="btn btn-accent" style={{ fontSize: '0.85rem', padding: '0.25rem 0.5rem' }}>
                      {unsuspendLabel}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
