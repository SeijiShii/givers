import { useState, useEffect, useRef } from "react";
import {
  getAdminThreads,
  createThread,
  broadcastThread,
  setThreadActive,
  searchUsers,
  getAdminUser,
  type MessageThread,
  type AdminUser,
} from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
}

export default function AdminMessageThreadList({ locale }: Props) {
  const [threads, setThreads] = useState<MessageThread[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);

  // Form state
  const [broadcast, setBroadcast] = useState(true);
  const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<AdminUser[]>([]);
  const [searching, setSearching] = useState(false);
  const [showSearchDropdown, setShowSearchDropdown] = useState(false);
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const searchWrapperRef = useRef<HTMLDivElement>(null);

  // Handle ?to= query param — fetch user by ID
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const toUserId = params.get("to");
    if (toUserId) {
      setBroadcast(false);
      setShowForm(true);
      getAdminUser(toUserId)
        .then((user) => {
          if (user) {
            setSelectedUser(user);
          }
        })
        .catch(() => {});
    }
  }, []);

  // Debounced incremental search
  useEffect(() => {
    if (broadcast || searchQuery.length < 2) {
      setSearchResults([]);
      setShowSearchDropdown(false);
      setSearching(false);
      return;
    }
    setSearching(true);
    setShowSearchDropdown(true);
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    searchTimerRef.current = setTimeout(() => {
      searchUsers(searchQuery)
        .then((users) => {
          setSearchResults(users);
          setShowSearchDropdown(true);
        })
        .catch(() => setSearchResults([]))
        .finally(() => setSearching(false));
    }, 300);
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, [searchQuery, broadcast]);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (
        searchWrapperRef.current &&
        !searchWrapperRef.current.contains(e.target as Node)
      ) {
        setShowSearchDropdown(false);
      }
    };
    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, []);

  const reload = () => {
    setLoading(true);
    getAdminThreads(50)
      .then((res) => setThreads(res.threads))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    reload();
  }, []);

  const handleCreate = async () => {
    if (!subject.trim() || !body.trim()) return;
    if (!broadcast && !selectedUser) return;
    setSubmitting(true);
    setError("");
    setSuccessMessage("");
    try {
      if (broadcast) {
        const count = await broadcastThread(subject.trim(), body.trim());
        setSuccessMessage(
          t(locale, "messages.broadcastSuccess") + ` (${count})`,
        );
      } else {
        await createThread(selectedUser!.id, subject.trim(), body.trim());
      }
      setShowForm(false);
      setSelectedUser(null);
      setSearchQuery("");
      setSubject("");
      setBody("");
      setBroadcast(true);
      reload();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleActive = (thread: MessageThread) => {
    setThreadActive(thread.id, !thread.active)
      .then(reload)
      .catch(() => {});
  };

  const handleSelectUser = (user: AdminUser) => {
    setSelectedUser(user);
    setSearchQuery("");
    setShowSearchDropdown(false);
  };

  const basePath = locale === "en" ? "/en" : "";

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
  };

  const canSubmit =
    !submitting &&
    subject.trim() &&
    body.trim() &&
    (broadcast || selectedUser !== null);

  return (
    <div className="admin-message-list">
      <div className="admin-header">
        <h1>{t(locale, "messages.manage")}</h1>
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => {
            setShowForm(!showForm);
            setSuccessMessage("");
          }}
        >
          {t(locale, "messages.newThread")}
        </button>
      </div>

      {successMessage && (
        <div className="success-banner">{successMessage}</div>
      )}

      {showForm && (
        <div className="admin-form card">
          <div className="form-group">
            <label className="broadcast-toggle">
              <input
                type="checkbox"
                checked={broadcast}
                onChange={(e) => {
                  setBroadcast(e.target.checked);
                  if (e.target.checked) {
                    setSelectedUser(null);
                    setSearchQuery("");
                  }
                }}
              />
              {t(locale, "messages.broadcastAll")}
            </label>
          </div>

          {!broadcast && (
            <div className="form-group">
              <label>{t(locale, "messages.recipient")}</label>
              {selectedUser ? (
                <div className="selected-user-chip">
                  <span>
                    {t(locale, "messages.selectedUser")}: <strong>{selectedUser.name}</strong> ({selectedUser.email})
                  </span>
                  <button
                    type="button"
                    className="chip-close-btn"
                    onClick={() => setSelectedUser(null)}
                  >
                    ×
                  </button>
                </div>
              ) : (
                <div ref={searchWrapperRef} className="user-search-wrapper">
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder={t(locale, "messages.searchUser")}
                  />
                  {showSearchDropdown && (
                    <div className="user-search-dropdown">
                      {searching ? (
                        <div className="search-status">
                          {t(locale, "messages.searching")}
                        </div>
                      ) : searchResults.length === 0 ? (
                        <div className="search-status">
                          {t(locale, "messages.noResults")}
                        </div>
                      ) : (
                        searchResults.map((user) => (
                          <button
                            key={user.id}
                            type="button"
                            className="user-search-item"
                            onClick={() => handleSelectUser(user)}
                          >
                            <strong>{user.name}</strong>{" "}
                            <span className="user-search-email">
                              {user.email}
                            </span>
                          </button>
                        ))
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          <div className="form-group">
            <label>{t(locale, "messages.subject")}</label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              maxLength={200}
            />
          </div>
          <div className="form-group">
            <label>{t(locale, "messages.body")}</label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={4}
            />
          </div>
          {error && <p className="error-message">{error}</p>}
          <div className="form-actions">
            <button
              type="button"
              className="btn btn-primary"
              onClick={handleCreate}
              disabled={!canSubmit}
            >
              {t(locale, "messages.send")}
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <p>...</p>
      ) : threads.length === 0 ? (
        <p className="empty-message">{t(locale, "messages.empty")}</p>
      ) : (
        <div className="thread-admin-list">
          {threads.map((thread) => (
            <div
              key={thread.id}
              className={`thread-admin-item card${!thread.active ? " inactive" : ""}`}
            >
              <div className="thread-admin-header">
                <strong>{thread.subject}</strong>
                <span className="thread-status">
                  {thread.active
                    ? t(locale, "messages.active")
                    : t(locale, "messages.inactive")}
                </span>
              </div>
              <div className="thread-admin-meta">
                <span>
                  {t(locale, "messages.recipient")}:{" "}
                  {thread.participant_name || thread.participant_user_id}
                </span>
                <span>{formatDate(thread.created_at)}</span>
              </div>
              {thread.last_message_body && (
                <div className="thread-preview">
                  {thread.last_message_body.length > 80
                    ? thread.last_message_body.slice(0, 80) + "\u2026"
                    : thread.last_message_body}
                </div>
              )}
              <div className="thread-admin-actions">
                <a href={`${basePath}/messages/${thread.id}`} className="btn btn-secondary">
                  {locale === "ja" ? "表示" : "View"}
                </a>
                <button
                  type="button"
                  className={`btn ${thread.active ? "btn-danger" : "btn-primary"}`}
                  onClick={() => handleToggleActive(thread)}
                >
                  {thread.active
                    ? t(locale, "messages.deactivate")
                    : t(locale, "messages.activate")}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
