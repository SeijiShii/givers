import { useState, useEffect, useRef, useCallback } from "react";
import {
  getMyThreads,
  getMessageUnreadCount,
  getMe,
  type MessageThread,
} from "../../lib/api";

interface Props {
  locale: string;
  basePath: string;
  titleLabel: string;
  emptyLabel: string;
  viewAllLabel: string;
}

export default function NavMessageBell({
  locale,
  basePath,
  titleLabel,
  emptyLabel,
  viewAllLabel,
}: Props) {
  const [unreadCount, setUnreadCount] = useState(0);
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<MessageThread[]>([]);
  const [loading, setLoading] = useState(false);
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getMe()
      .then((user) => setLoggedIn(!!user))
      .catch(() => setLoggedIn(false));
  }, []);

  useEffect(() => {
    if (loggedIn !== true) return;
    getMessageUnreadCount()
      .then(setUnreadCount)
      .catch(() => {});
  }, [loggedIn]);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, []);

  const toggle = useCallback(() => {
    if (!open) {
      setLoading(true);
      getMyThreads(5)
        .then((res) => setItems(res.threads))
        .catch(() => {})
        .finally(() => setLoading(false));
    }
    setOpen((v) => !v);
  }, [open]);

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return `${d.getMonth() + 1}/${d.getDate()}`;
  };

  if (loggedIn !== true) return null;

  return (
    <div ref={ref} style={{ position: "relative", display: "inline-block" }}>
      <button
        type="button"
        onClick={toggle}
        aria-label={titleLabel}
        className="message-bell-btn"
      >
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
        {unreadCount > 0 && (
          <span className="message-badge">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="announcement-dropdown">
          <div className="announcement-dropdown-header">
            <strong>{titleLabel}</strong>
            <a href={`${basePath}/messages`}>{viewAllLabel}</a>
          </div>
          {loading ? (
            <div className="announcement-dropdown-loading">...</div>
          ) : items.length === 0 ? (
            <div className="announcement-dropdown-empty">{emptyLabel}</div>
          ) : (
            <div className="announcement-dropdown-list">
              {items.map((item) => (
                <a
                  key={item.id}
                  href={`${basePath}/messages/${item.id}`}
                  className={`announcement-dropdown-item${(item.unread_count ?? 0) > 0 ? " unread" : ""}`}
                  style={{ textDecoration: "none", color: "inherit" }}
                >
                  <div className="announcement-dropdown-item-content">
                    <div className="announcement-dropdown-item-title">
                      {item.subject}
                    </div>
                    <div className="announcement-dropdown-item-body">
                      {(item.last_message_body ?? "").length > 40
                        ? (item.last_message_body ?? "").slice(0, 40) + "\u2026"
                        : item.last_message_body}
                    </div>
                  </div>
                  <span className="announcement-dropdown-item-date">
                    {item.last_message_at
                      ? formatDate(item.last_message_at)
                      : ""}
                  </span>
                </a>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
