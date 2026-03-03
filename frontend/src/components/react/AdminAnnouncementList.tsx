import { useState, useEffect } from "react";
import {
  getAdminAnnouncements,
  createAnnouncement,
  updateAnnouncement,
  setAnnouncementVisibility,
  type Announcement,
} from "../../lib/api";

interface Props {
  locale: string;
  title: string;
  forbiddenMessage: string;
  createLabel: string;
  editLabel: string;
  titleLabel: string;
  bodyLabel: string;
  severityLabel: string;
  publishLabel: string;
  hideLabel: string;
  submitLabel: string;
  submitScheduledLabel: string;
  updateLabel: string;
  scheduleLabel: string;
  scheduledAtLabel: string;
  statusPublished: string;
  statusScheduled: string;
  statusHidden: string;
}

const SEVERITY_ICON: Record<string, { icon: string; color: string }> = {
  info: { icon: "\u24D8", color: "#3b82f6" },
  warn: { icon: "\u26A0", color: "#f59e0b" },
  error: { icon: "\u26D4", color: "#ef4444" },
};

export default function AdminAnnouncementList(props: Props) {
  const [items, setItems] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Announcement | null>(null);
  const [showForm, setShowForm] = useState(false);

  // Form state
  const [formTitle, setFormTitle] = useState("");
  const [formBody, setFormBody] = useState("");
  const [formSeverity, setFormSeverity] = useState("info");
  const [scheduleEnabled, setScheduleEnabled] = useState(false);
  const [formPublishedAt, setFormPublishedAt] = useState("");

  const reload = () => {
    setLoading(true);
    getAdminAnnouncements(50)
      .then((res) => setItems(res.announcements))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    reload();
  }, []);

  const resetForm = () => {
    setFormTitle("");
    setFormBody("");
    setFormSeverity("info");
    setScheduleEnabled(false);
    setFormPublishedAt("");
    setEditing(null);
    setShowForm(false);
  };

  const startEdit = (a: Announcement) => {
    setEditing(a);
    setFormTitle(a.title);
    setFormBody(a.body);
    setFormSeverity(a.severity);
    const isFuture = new Date(a.published_at) > new Date();
    setScheduleEnabled(isFuture);
    if (isFuture) {
      setFormPublishedAt(a.published_at.slice(0, 16));
    }
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formTitle.trim()) return;

    const input: {
      title: string;
      body?: string;
      severity?: string;
      published_at?: string;
    } = {
      title: formTitle.trim(),
      body: formBody,
      severity: formSeverity,
    };
    if (scheduleEnabled && formPublishedAt) {
      input.published_at = new Date(formPublishedAt).toISOString();
    }

    try {
      if (editing) {
        await updateAnnouncement(editing.id, input);
      } else {
        await createAnnouncement(input);
      }
      resetForm();
      reload();
    } catch {
      // error handling could be added
    }
  };

  const toggleVisibility = async (a: Announcement) => {
    await setAnnouncementVisibility(a.id, !a.visible).catch(() => {});
    reload();
  };

  const getStatus = (a: Announcement) => {
    if (!a.visible) return { label: props.statusHidden, className: "status-hidden" };
    if (new Date(a.published_at) > new Date())
      return { label: `\uD83D\uDD50 ${props.statusScheduled}`, className: "status-scheduled" };
    return { label: props.statusPublished, className: "status-published" };
  };

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
  };

  return (
    <div>
      <h1>{props.title}</h1>

      {!showForm && (
        <button
          type="button"
          className="btn btn-accent"
          onClick={() => {
            resetForm();
            setShowForm(true);
          }}
          style={{ marginBottom: "1rem" }}
        >
          + {props.createLabel}
        </button>
      )}

      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="announcement-form"
          style={{
            marginBottom: "1.5rem",
            padding: "1rem",
            border: "1px solid var(--color-border, #ddd)",
            borderRadius: "8px",
          }}
        >
          <h3>{editing ? props.editLabel : props.createLabel}</h3>
          <div style={{ marginBottom: "0.5rem" }}>
            <label>
              {props.titleLabel}
              <input
                type="text"
                value={formTitle}
                onChange={(e) => setFormTitle(e.target.value)}
                maxLength={200}
                required
                style={{ width: "100%", marginTop: "0.25rem" }}
              />
            </label>
          </div>
          <div style={{ marginBottom: "0.5rem" }}>
            <label>
              {props.bodyLabel}
              <textarea
                value={formBody}
                onChange={(e) => setFormBody(e.target.value)}
                rows={4}
                style={{ width: "100%", marginTop: "0.25rem" }}
              />
            </label>
          </div>
          <div style={{ marginBottom: "0.5rem" }}>
            <label>
              {props.severityLabel}
              <select
                value={formSeverity}
                onChange={(e) => setFormSeverity(e.target.value)}
                style={{ marginLeft: "0.5rem" }}
              >
                <option value="info">INFO</option>
                <option value="warn">WARN</option>
                <option value="error">ERROR</option>
              </select>
            </label>
          </div>
          <div style={{ marginBottom: "0.5rem" }}>
            <label>
              <input
                type="checkbox"
                checked={scheduleEnabled}
                onChange={(e) => setScheduleEnabled(e.target.checked)}
              />{" "}
              {props.scheduleLabel}
            </label>
            {scheduleEnabled && (
              <div style={{ marginTop: "0.25rem" }}>
                <label>
                  {props.scheduledAtLabel}
                  <input
                    type="datetime-local"
                    value={formPublishedAt}
                    onChange={(e) => setFormPublishedAt(e.target.value)}
                    style={{ marginLeft: "0.5rem" }}
                  />
                </label>
              </div>
            )}
          </div>
          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button type="submit" className="btn btn-accent">
              {editing
                ? props.updateLabel
                : scheduleEnabled
                  ? props.submitScheduledLabel
                  : props.submitLabel}
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={resetForm}
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : (
        <table className="admin-table" style={{ width: "100%" }}>
          <thead>
            <tr>
              <th></th>
              <th>{props.titleLabel}</th>
              <th>Status</th>
              <th>Date</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const sev = SEVERITY_ICON[item.severity] ?? SEVERITY_ICON.info;
              const status = getStatus(item);
              return (
                <tr key={item.id}>
                  <td style={{ color: sev.color, fontSize: "1.2rem" }}>
                    {sev.icon}
                  </td>
                  <td>{item.title}</td>
                  <td>
                    <span className={status.className}>{status.label}</span>
                  </td>
                  <td>{formatDate(item.published_at)}</td>
                  <td>
                    <button
                      type="button"
                      className="btn btn-sm"
                      onClick={() => startEdit(item)}
                    >
                      {props.editLabel}
                    </button>
                    <button
                      type="button"
                      className="btn btn-sm"
                      onClick={() => toggleVisibility(item)}
                      style={{ marginLeft: "0.5rem" }}
                    >
                      {item.visible ? props.hideLabel : props.publishLabel}
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}
