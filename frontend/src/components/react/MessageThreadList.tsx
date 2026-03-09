import { useState, useEffect } from "react";
import { getMyThreads, getMe, setThreadActive, type MessageThread } from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
  basePath: string;
}

export default function MessageThreadList({ locale, basePath }: Props) {
  const [threads, setThreads] = useState<MessageThread[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [isHost, setIsHost] = useState(false);

  const load = (cursor?: string) => {
    setLoading(true);
    getMyThreads(20, cursor)
      .then((res) => {
        setThreads((prev) =>
          cursor ? [...prev, ...res.threads] : res.threads,
        );
        setNextCursor(res.next_cursor);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    getMe()
      .then((user) => setIsHost(user?.role === "host"))
      .catch(() => {});
  }, []);

  const handleToggleActive = (thread: MessageThread) => {
    setThreadActive(thread.id, !thread.active)
      .then(() => {
        setThreads((prev) =>
          prev.map((t) =>
            t.id === thread.id ? { ...t, active: !t.active } : t,
          ),
        );
      })
      .catch(() => {});
  };

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
  };

  return (
    <div className="message-thread-list">
      <h1>{t(locale, "messages.title")}</h1>

      {loading && threads.length === 0 ? (
        <p>...</p>
      ) : threads.length === 0 ? (
        <p className="empty-message">{t(locale, "messages.empty")}</p>
      ) : (
        <>
          <div className="thread-list">
            {threads.map((thread) => (
              <div
                key={thread.id}
                className={`thread-item${(thread.unread_count ?? 0) > 0 ? " unread" : ""}${!thread.active ? " inactive" : ""}`}
              >
                <a
                  href={`${basePath}/messages/${thread.id}`}
                  className="thread-item-link"
                >
                  <div className="thread-item-header">
                    <span className="thread-subject">{thread.subject}</span>
                    <span className="thread-status">
                      {thread.active
                        ? t(locale, "messages.active")
                        : t(locale, "messages.inactive")}
                    </span>
                  </div>
                  <div className="thread-item-meta">
                    <span className="thread-participant">
                      {thread.host_name || thread.participant_name}
                    </span>
                    <span className="thread-date">
                      {thread.last_message_at
                        ? formatDate(thread.last_message_at)
                        : formatDate(thread.created_at)}
                    </span>
                  </div>
                  <div className="thread-preview">
                    {(thread.last_message_body ?? "").length > 60
                      ? (thread.last_message_body ?? "").slice(0, 60) + "\u2026"
                      : thread.last_message_body}
                  </div>
                  {(thread.unread_count ?? 0) > 0 && (
                    <span className="thread-unread-badge">
                      {thread.unread_count}
                    </span>
                  )}
                </a>
                {isHost && (
                  <div className="thread-item-actions">
                    <button
                      type="button"
                      className={`btn btn-sm ${thread.active ? "btn-secondary" : "btn-primary"}`}
                      onClick={() => handleToggleActive(thread)}
                    >
                      {thread.active
                        ? t(locale, "messages.deactivate")
                        : t(locale, "messages.activate")}
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>

          {nextCursor && (
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => load(nextCursor)}
              disabled={loading}
            >
              {t(locale, "messages.loadMore")}
            </button>
          )}
        </>
      )}
    </div>
  );
}
