import { useState, useEffect } from "react";
import {
  getAnnouncements,
  markAnnouncementRead,
  getMe,
  type Announcement,
} from "../../lib/api";

interface Props {
  locale: string;
  basePath: string;
  title: string;
  emptyLabel: string;
  loadMoreLabel: string;
  manageLabel: string;
}

const SEVERITY_ICON: Record<string, { icon: string; color: string }> = {
  info: { icon: "\u24D8", color: "#3b82f6" },
  warn: { icon: "\u26A0", color: "#f59e0b" },
  error: { icon: "\u26D4", color: "#ef4444" },
};

export default function AnnouncementList({
  locale,
  basePath,
  title,
  emptyLabel,
  loadMoreLabel,
  manageLabel,
}: Props) {
  const [items, setItems] = useState<Announcement[]>([]);
  const [cursor, setCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [isHost, setIsHost] = useState(false);

  useEffect(() => {
    getMe()
      .then((user) => setIsHost(user?.role === "host"))
      .catch(() => {});
  }, []);

  const load = (c?: string) => {
    setLoading(true);
    getAnnouncements(20, c ?? undefined)
      .then((res) => {
        const newItems = c
          ? [...items, ...res.announcements]
          : res.announcements;
        setItems(newItems);
        setCursor(res.next_cursor);
        // Mark unread as read (skip for hosts — they post announcements)
        if (!isHost) {
          res.announcements
            .filter((a) => a.is_read === false)
            .forEach((a) => markAnnouncementRead(a.id).catch(() => {}));
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    if (locale === "en") {
      return d.toLocaleDateString("en-US", {
        year: "numeric",
        month: "long",
        day: "numeric",
      });
    }
    return `${d.getFullYear()}\u5E74${d.getMonth() + 1}\u6708${d.getDate()}\u65E5`;
  };

  return (
    <div className="announcement-page">
      <div className="announcement-page-header">
        <h1>{title}</h1>
        {isHost && (
          <a
            href={`${basePath}/admin/announcements`}
            className="btn btn-secondary btn-sm"
          >
            {manageLabel}
          </a>
        )}
      </div>
      {!loading && items.length === 0 && (
        <p className="announcement-empty">{emptyLabel}</p>
      )}
      <div className="announcement-list">
        {items.map((item) => {
          const sev = SEVERITY_ICON[item.severity] ?? SEVERITY_ICON.info;
          return (
            <div
              key={item.id}
              className={`announcement-card${!isHost && item.is_read === false ? " unread" : ""}`}
            >
              <div className="announcement-card-header">
                <span
                  className="announcement-severity-icon"
                  style={{ color: sev.color }}
                >
                  {sev.icon}
                </span>
                <span className="announcement-card-title">{item.title}</span>
                <span className="announcement-card-date">
                  {formatDate(item.published_at)}
                </span>
              </div>
              {item.body && (
                <div className="announcement-card-body">{item.body}</div>
              )}
            </div>
          );
        })}
      </div>
      {loading && <p style={{ textAlign: "center" }}>Loading...</p>}
      {cursor && !loading && (
        <div style={{ textAlign: "center", marginTop: "1rem" }}>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => load(cursor)}
          >
            {loadMoreLabel}
          </button>
        </div>
      )}
    </div>
  );
}
