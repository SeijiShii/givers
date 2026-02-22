import { useEffect, useState } from "react";
import {
  getMe,
  getAdminUsers,
  suspendUser,
  getDisclosureExport,
  type User,
  type AdminUser,
} from "../../lib/api";
import type { Locale } from "../../lib/i18n";
import { t } from "../../lib/i18n";

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
  const [suspendingId, setSuspendingId] = useState<string | null>(null);
  const [disclosureType, setDisclosureType] = useState<"user" | "project">(
    "user",
  );
  const [disclosureId, setDisclosureId] = useState("");
  const [disclosureLoading, setDisclosureLoading] = useState(false);
  const [disclosureConfirmOpen, setDisclosureConfirmOpen] = useState(false);
  const [disclosureError, setDisclosureError] = useState<string | null>(null);

  useEffect(() => {
    getMe()
      .then(setMe)
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (me?.role === "host") {
      getAdminUsers()
        .then(setUsers)
        .catch(() => setUsers([]));
    }
  }, [me?.role]);

  if (loading) {
    return <p>{t(locale, "projects.loading")}</p>;
  }

  const handleDisclosureExport = async () => {
    const id = disclosureId.trim();
    if (!id) return;
    setDisclosureError(null);
    setDisclosureLoading(true);
    try {
      const data = await getDisclosureExport(disclosureType, id);
      const blob = new Blob([JSON.stringify(data, null, 2)], {
        type: "application/json",
      });
      const name = `disclosure-export-${disclosureType}-${id}-${new Date().toISOString().slice(0, 10)}.json`;
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = name;
      a.click();
      URL.revokeObjectURL(a.href);
      setDisclosureConfirmOpen(false);
      setDisclosureId("");
    } catch (e) {
      setDisclosureError(e instanceof Error ? e.message : "Export failed");
    } finally {
      setDisclosureLoading(false);
    }
  };

  if (!me || me.role !== "host") {
    return (
      <div className="card" style={{ padding: "1.5rem" }}>
        <p style={{ color: "var(--color-text-muted)", margin: 0 }}>
          {forbiddenMessage}
        </p>
      </div>
    );
  }

  return (
    <div className="admin-users">
      <h1>{title}</h1>
      <div className="card" style={{ marginTop: "1rem", padding: "1rem" }}>
        <h2 style={{ margin: "0 0 0.5rem", fontSize: "1rem" }}>
          {t(locale, "admin.disclosureExportTitle")}
        </h2>
        <p
          style={{
            margin: "0 0 1rem",
            fontSize: "0.9rem",
            color: "var(--color-text-muted)",
          }}
        >
          {t(locale, "admin.disclosureExportDescription")}
        </p>
        <div
          style={{
            display: "flex",
            flexWrap: "wrap",
            gap: "0.75rem",
            alignItems: "center",
          }}
        >
          <select
            value={disclosureType}
            onChange={(e) =>
              setDisclosureType(e.target.value as "user" | "project")
            }
            style={{
              padding: "0.35rem 0.5rem",
              borderRadius: "4px",
              border: "1px solid var(--color-border)",
            }}
          >
            <option value="user">
              {t(locale, "admin.disclosureExportTypeUser")}
            </option>
            <option value="project">
              {t(locale, "admin.disclosureExportTypeProject")}
            </option>
          </select>
          <input
            type="text"
            value={disclosureId}
            onChange={(e) => setDisclosureId(e.target.value)}
            placeholder={t(locale, "admin.disclosureExportIdPlaceholder")}
            style={{
              padding: "0.35rem 0.5rem",
              minWidth: "140px",
              borderRadius: "4px",
              border: "1px solid var(--color-border)",
            }}
          />
          <button
            type="button"
            className="btn btn-accent"
            disabled={!disclosureId.trim() || disclosureLoading}
            onClick={() => setDisclosureConfirmOpen(true)}
          >
            {disclosureLoading
              ? locale === "ja"
                ? "出力中..."
                : "Exporting..."
              : t(locale, "admin.disclosureExportButton")}
          </button>
        </div>
        {disclosureError && (
          <p
            style={{
              margin: "0.5rem 0 0",
              fontSize: "0.85rem",
              color: "var(--color-danger)",
            }}
          >
            {disclosureError}
          </p>
        )}
      </div>
      {disclosureConfirmOpen && (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: "fixed",
            inset: 0,
            zIndex: 1000,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            background: "rgba(0,0,0,0.5)",
          }}
          onClick={() => !disclosureLoading && setDisclosureConfirmOpen(false)}
        >
          <div
            style={{
              background: "var(--color-bg)",
              padding: "1.5rem",
              borderRadius: "8px",
              maxWidth: "400px",
              boxShadow: "0 4px 20px rgba(0,0,0,0.15)",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <p style={{ margin: "0 0 1rem", fontSize: "0.9rem" }}>
              {t(locale, "admin.disclosureExportNotice")}
            </p>
            <div
              style={{
                display: "flex",
                gap: "0.5rem",
                justifyContent: "flex-end",
              }}
            >
              <button
                type="button"
                className="btn"
                onClick={() =>
                  !disclosureLoading && setDisclosureConfirmOpen(false)
                }
                disabled={disclosureLoading}
              >
                {locale === "ja" ? "キャンセル" : "Cancel"}
              </button>
              <button
                type="button"
                className="btn btn-accent"
                onClick={handleDisclosureExport}
                disabled={disclosureLoading}
              >
                {t(locale, "admin.disclosureExportConfirm")}
              </button>
            </div>
          </div>
        </div>
      )}
      <div className="card" style={{ marginTop: "1rem", overflowX: "auto" }}>
        <table style={{ width: "100%", borderCollapse: "collapse" }}>
          <thead>
            <tr style={{ borderBottom: "2px solid var(--color-border)" }}>
              <th style={{ padding: "0.75rem", textAlign: "left" }}>
                {locale === "ja" ? "名前" : "Name"}
              </th>
              <th style={{ padding: "0.75rem", textAlign: "left" }}>
                {locale === "ja" ? "メール" : "Email"}
              </th>
              <th style={{ padding: "0.75rem", textAlign: "left" }}>
                {locale === "ja" ? "ステータス" : "Status"}
              </th>
              <th style={{ padding: "0.75rem", textAlign: "right" }}>
                {projectCountLabel}
              </th>
              <th style={{ padding: "0.75rem" }}>
                {locale === "ja" ? "操作" : "Actions"}
              </th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr
                key={u.id}
                style={{ borderBottom: "1px solid var(--color-border-light)" }}
              >
                <td style={{ padding: "0.75rem" }}>{u.name}</td>
                <td style={{ padding: "0.75rem", fontSize: "0.9rem" }}>
                  {u.email}
                </td>
                <td style={{ padding: "0.75rem" }}>
                  <span
                    style={{
                      padding: "0.2rem 0.5rem",
                      borderRadius: "4px",
                      fontSize: "0.85rem",
                      backgroundColor:
                        u.status === "active"
                          ? "var(--color-primary-muted)"
                          : "var(--color-danger)",
                      color:
                        u.status === "active" ? "var(--color-text)" : "white",
                    }}
                  >
                    {u.status === "active" ? statusActive : statusSuspended}
                  </span>
                </td>
                <td style={{ padding: "0.75rem", textAlign: "right" }}>
                  {u.project_count ?? 0}
                </td>
                <td style={{ padding: "0.75rem" }}>
                  <span
                    style={{
                      display: "flex",
                      gap: "0.35rem",
                      flexWrap: "wrap",
                    }}
                  >
                    {u.id === me?.id ? null : u.status === "active" ? (
                      <button
                        type="button"
                        className="btn"
                        disabled={suspendingId === u.id}
                        style={{
                          fontSize: "0.85rem",
                          padding: "0.25rem 0.5rem",
                        }}
                        onClick={async () => {
                          setSuspendingId(u.id);
                          try {
                            await suspendUser(u.id, true);
                            setUsers((prev) =>
                              prev.map((x) =>
                                x.id === u.id
                                  ? { ...x, status: "suspended" as const }
                                  : x,
                              ),
                            );
                          } catch {
                            /* ignore */
                          }
                          setSuspendingId(null);
                        }}
                      >
                        {suspendLabel}
                      </button>
                    ) : (
                      <button
                        type="button"
                        className="btn btn-accent"
                        disabled={suspendingId === u.id}
                        style={{
                          fontSize: "0.85rem",
                          padding: "0.25rem 0.5rem",
                        }}
                        onClick={async () => {
                          setSuspendingId(u.id);
                          try {
                            await suspendUser(u.id, false);
                            setUsers((prev) =>
                              prev.map((x) =>
                                x.id === u.id
                                  ? { ...x, status: "active" as const }
                                  : x,
                              ),
                            );
                          } catch {
                            /* ignore */
                          }
                          setSuspendingId(null);
                        }}
                      >
                        {unsuspendLabel}
                      </button>
                    )}
                    <button
                      type="button"
                      className="btn"
                      style={{ fontSize: "0.8rem", padding: "0.2rem 0.4rem" }}
                      onClick={() => {
                        setDisclosureType("user");
                        setDisclosureId(u.id);
                        setDisclosureConfirmOpen(true);
                      }}
                    >
                      {locale === "ja" ? "開示用" : "Export"}
                    </button>
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
