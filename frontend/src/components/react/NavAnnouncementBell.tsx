import { useState, useEffect, useRef, useCallback } from "react";
import {
  getAnnouncements,
  getAnnouncementUnreadCount,
  markAnnouncementRead,
  getMe,
  type Announcement,
} from "../../lib/api";

interface Props {
  locale: string;
  basePath: string;
  viewAllLabel: string;
  titleLabel: string;
  emptyLabel: string;
}

const SEVERITY_ICON: Record<string, { icon: string; color: string }> = {
  info: { icon: "\u24D8", color: "#3b82f6" },
  warn: { icon: "\u26A0", color: "#f59e0b" },
  error: { icon: "\u26D4", color: "#ef4444" },
};

export default function NavAnnouncementBell({
  locale,
  basePath,
  viewAllLabel,
  titleLabel,
  emptyLabel,
}: Props) {
  const [unreadCount, setUnreadCount] = useState(0);
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(false);
  const [isHost, setIsHost] = useState<boolean | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  // Check if user is host
  useEffect(() => {
    getMe()
      .then((user) => setIsHost(user?.role === "host"))
      .catch(() => setIsHost(false));
  }, []);

  // Fetch unread count on mount (skip for hosts)
  useEffect(() => {
    if (isHost !== false) return;
    getAnnouncementUnreadCount()
      .then(setUnreadCount)
      .catch(() => {});
  }, [isHost]);

  // Close on outside click
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
      getAnnouncements(5)
        .then((res) => {
          setItems(res.announcements);
          // Mark visible unread as read
          const unread = res.announcements.filter((a) => a.is_read === false);
          unread.forEach((a) => markAnnouncementRead(a.id).catch(() => {}));
          if (unread.length > 0) {
            setUnreadCount((c) => Math.max(0, c - unread.length));
          }
        })
        .catch(() => {})
        .finally(() => setLoading(false));
    }
    setOpen((v) => !v);
  }, [open]);

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return `${d.getMonth() + 1}/${d.getDate()}`;
  };

  // Host posts announcements — no bell needed
  if (isHost === true) return null;

  return (
    <div ref={ref} style={{ position: "relative", display: "inline-block" }}>
      <button
        type="button"
        onClick={toggle}
        aria-label={titleLabel}
        className="announcement-bell-btn"
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
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" />
        </svg>
        {unreadCount > 0 && (
          <span className="announcement-badge">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="announcement-dropdown">
          <div className="announcement-dropdown-header">
            <strong>{titleLabel}</strong>
            <a href={`${basePath}/announcements`}>{viewAllLabel}</a>
          </div>
          {loading ? (
            <div className="announcement-dropdown-loading">...</div>
          ) : items.length === 0 ? (
            <div className="announcement-dropdown-empty">{emptyLabel}</div>
          ) : (
            <div className="announcement-dropdown-list">
              {items.map((item) => {
                const sev = SEVERITY_ICON[item.severity] ?? SEVERITY_ICON.info;
                return (
                  <div
                    key={item.id}
                    className={`announcement-dropdown-item${item.is_read === false ? " unread" : ""}`}
                  >
                    <span
                      className="announcement-severity-icon"
                      style={{ color: sev.color }}
                    >
                      {sev.icon}
                    </span>
                    <div className="announcement-dropdown-item-content">
                      <div className="announcement-dropdown-item-title">
                        {item.title}
                      </div>
                      <div className="announcement-dropdown-item-body">
                        {item.body.length > 50
                          ? item.body.slice(0, 50) + "\u2026"
                          : item.body}
                      </div>
                    </div>
                    <span className="announcement-dropdown-item-date">
                      {formatDate(item.published_at)}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
