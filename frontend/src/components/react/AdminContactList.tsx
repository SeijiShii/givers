import { useEffect, useState } from 'react';
import { getMe, getAdminContacts, markContactRead, type User, type ContactMessage } from '../../lib/api';

interface Props {
  locale: string;
  title: string;
  forbiddenMessage: string;
  emptyLabel: string;
  statusUnreadLabel: string;
  statusReadLabel: string;
  markReadLabel: string;
  fromLabel: string;
  dateLabel: string;
  messageLabel: string;
  filterAllLabel: string;
  filterUnreadLabel: string;
  loadingLabel: string;
}

export default function AdminContactList({
  locale,
  title,
  forbiddenMessage,
  emptyLabel,
  statusUnreadLabel,
  statusReadLabel,
  markReadLabel,
  fromLabel,
  dateLabel,
  messageLabel,
  filterAllLabel,
  filterUnreadLabel,
  loadingLabel,
}: Props) {
  const [me, setMe] = useState<User | null>(null);
  const [messages, setMessages] = useState<ContactMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');
  const [markingIds, setMarkingIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    getMe()
      .then(setMe)
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (me?.role !== 'host') return;
    setLoading(true);
    getAdminContacts({ status: filter === 'unread' ? 'unread' : 'all' })
      .then(setMessages)
      .catch(() => setMessages([]))
      .finally(() => setLoading(false));
  }, [me?.role, filter]);

  const handleMarkRead = async (id: string) => {
    setMarkingIds((prev) => new Set(prev).add(id));
    try {
      await markContactRead(id);
      setMessages((prev) =>
        prev.map((m) => (m.id === id ? { ...m, status: 'read' as const } : m)),
      );
    } finally {
      setMarkingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  };

  if (loading) {
    return <p>{loadingLabel}</p>;
  }

  if (me?.role !== 'host') {
    return (
      <div className="card" style={{ marginTop: '1rem', padding: '1rem' }}>
        <p style={{ color: 'var(--color-danger, #c00)' }}>{forbiddenMessage}</p>
      </div>
    );
  }

  const unreadCount = messages.filter((m) => m.status === 'unread').length;

  return (
    <div style={{ marginTop: '1rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
        <h1 style={{ margin: 0 }}>
          {title}
          {unreadCount > 0 && filter === 'all' && (
            <span
              style={{
                marginLeft: '0.5rem',
                fontSize: '0.8rem',
                background: 'var(--color-danger, #c00)',
                color: 'white',
                borderRadius: '9999px',
                padding: '0.1rem 0.5rem',
                verticalAlign: 'middle',
              }}
            >
              {unreadCount}
            </span>
          )}
        </h1>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button
            type="button"
            className={`btn${filter === 'all' ? ' btn-primary' : ''}`}
            onClick={() => setFilter('all')}
          >
            {filterAllLabel}
          </button>
          <button
            type="button"
            className={`btn${filter === 'unread' ? ' btn-primary' : ''}`}
            onClick={() => setFilter('unread')}
          >
            {filterUnreadLabel}
          </button>
        </div>
      </div>

      {messages.length === 0 ? (
        <div className="card" style={{ padding: '1.5rem', color: 'var(--color-text-muted, #888)' }}>
          {emptyLabel}
        </div>
      ) : (
        <div style={{ display: 'grid', gap: '0.75rem' }}>
          {messages.map((msg) => (
            <div
              key={msg.id}
              className="card"
              style={{
                padding: '1rem 1.25rem',
                borderLeft: msg.status === 'unread' ? '4px solid var(--color-primary, #007bff)' : '4px solid transparent',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '0.5rem', flexWrap: 'wrap' }}>
                <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', fontSize: '0.9rem', color: 'var(--color-text-muted, #555)' }}>
                  <span>
                    <strong>{fromLabel}:</strong>{' '}
                    {msg.name ? `${msg.name} ` : ''}
                    <a href={`mailto:${msg.email}`}>{msg.email}</a>
                  </span>
                  <span>
                    <strong>{dateLabel}:</strong>{' '}
                    {new Date(msg.created_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')}
                  </span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                  <span
                    style={{
                      fontSize: '0.75rem',
                      padding: '0.1rem 0.5rem',
                      borderRadius: '4px',
                      background: msg.status === 'unread' ? 'var(--color-primary, #007bff)' : 'var(--color-muted, #eee)',
                      color: msg.status === 'unread' ? 'white' : 'var(--color-text-muted, #555)',
                    }}
                  >
                    {msg.status === 'unread' ? statusUnreadLabel : statusReadLabel}
                  </span>
                  {msg.status === 'unread' && (
                    <button
                      type="button"
                      className="btn"
                      style={{ fontSize: '0.8rem', padding: '0.15rem 0.5rem' }}
                      disabled={markingIds.has(msg.id)}
                      onClick={() => handleMarkRead(msg.id)}
                    >
                      {markReadLabel}
                    </button>
                  )}
                </div>
              </div>
              <p style={{ margin: '0.75rem 0 0', whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>
                {msg.message}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
