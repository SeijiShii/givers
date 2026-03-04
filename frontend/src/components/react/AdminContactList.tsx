import { useEffect, useState } from 'react';
import { getMe, getAdminContacts, markContactRead, markContactUnread, type User, type ContactMessage } from '../../lib/api';
import type { Locale } from '../../lib/i18n';

const PAGE_SIZE = 20;

interface Props {
  locale: Locale;
  title: string;
  forbiddenMessage: string;
  emptyLabel: string;
  statusUnreadLabel: string;
  statusReadLabel: string;
  markReadLabel: string;
  markUnreadLabel: string;
  fromLabel: string;
  dateLabel: string;
  messageLabel: string;
  filterAllLabel: string;
  filterUnreadLabel: string;
  loadingLabel: string;
  loadMoreLabel: string;
}

export default function AdminContactList({
  locale,
  title,
  forbiddenMessage,
  emptyLabel,
  statusUnreadLabel,
  statusReadLabel,
  markReadLabel,
  markUnreadLabel,
  fromLabel,
  dateLabel,
  messageLabel,
  filterAllLabel,
  filterUnreadLabel,
  loadingLabel,
  loadMoreLabel,
}: Props) {
  const [me, setMe] = useState<User | null>(null);
  const [messages, setMessages] = useState<ContactMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');
  const [markingIds, setMarkingIds] = useState<Set<string>>(new Set());
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  useEffect(() => {
    getMe()
      .then(setMe)
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, []);

  const fetchMessages = (status: 'all' | 'unread', off: number, append: boolean) => {
    getAdminContacts({
      status: status === 'all' ? undefined : 'unread',
      limit: PAGE_SIZE,
      offset: off,
    })
      .then((data) => {
        setMessages((prev) => (append ? [...prev, ...data] : data));
        setHasMore(data.length === PAGE_SIZE);
      })
      .catch(() => {
        if (!append) setMessages([]);
      });
  };

  useEffect(() => {
    if (me?.role !== 'host') return;
    setOffset(0);
    fetchMessages(filter, 0, false);
  }, [me?.role, filter]);

  const handleToggleStatus = async (msg: ContactMessage) => {
    setMarkingIds((prev) => new Set(prev).add(msg.id));
    try {
      if (msg.status === 'unread') {
        await markContactRead(msg.id);
        setMessages((prev) =>
          prev.map((m) => (m.id === msg.id ? { ...m, status: 'read' as const } : m)),
        );
      } else {
        await markContactUnread(msg.id);
        setMessages((prev) =>
          prev.map((m) => (m.id === msg.id ? { ...m, status: 'unread' as const } : m)),
        );
      }
    } finally {
      setMarkingIds((prev) => {
        const next = new Set(prev);
        next.delete(msg.id);
        return next;
      });
    }
  };

  const handleLoadMore = () => {
    const newOffset = offset + PAGE_SIZE;
    setOffset(newOffset);
    fetchMessages(filter, newOffset, true);
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
                  <button
                    type="button"
                    className="btn"
                    style={{ fontSize: '0.8rem', padding: '0.15rem 0.5rem' }}
                    disabled={markingIds.has(msg.id)}
                    onClick={() => handleToggleStatus(msg)}
                  >
                    {msg.status === 'unread' ? markReadLabel : markUnreadLabel}
                  </button>
                </div>
              </div>
              <p style={{ margin: '0.75rem 0 0', whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>
                {msg.message}
              </p>
            </div>
          ))}
        </div>
      )}

      {hasMore && (
        <div style={{ textAlign: 'center', marginTop: '1rem' }}>
          <button type="button" className="btn" onClick={handleLoadMore}>
            {loadMoreLabel}
          </button>
        </div>
      )}
    </div>
  );
}
