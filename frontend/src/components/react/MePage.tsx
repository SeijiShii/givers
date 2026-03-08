import { useEffect, useState } from "react";
import {
  getMe,
  getMyDonations,
  getMySubscriptions,
  getMyProjects,
  getWatchedProjects,
  unwatchProject,
  updateProject,
  updateSubscription,
  pauseSubscription,
  resumeSubscription,
  deleteSubscription,
  refundDonationByDonor,
  quickDonate,
  type User,
  type Donation,
  type RecurringDonation,
  type Project,
} from "../../lib/api";
import ConfirmDialog from "./ConfirmDialog";
import LoadingSkeleton from "./LoadingSkeleton";
import type { Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
  tabDonationsLabel: string;
  tabProjectsLabel: string;
  donationHistoryLabel: string;
  recurringDonationsLabel: string;
  noDonationsLabel: string;
  noRecurringLabel: string;
  cancelRecurringLabel: string;
  cancelledLabel: string;
  editRecurringLabel: string;
  pauseRecurringLabel: string;
  resumeRecurringLabel: string;
  deleteRecurringLabel: string;
  pausedLabel: string;
  intervalMonthlyLabel: string;
  intervalYearlyLabel: string;
  intervalLabel: string;
  editRecurringTitle: string;
  deleteRecurringConfirmTitle: string;
  deleteRecurringConfirmLabel: string;
  saveLabel: string;
  cancelLabel: string;
  projectNameLabel: string;
  amountLabel: string;
  dateLabel: string;
  messageLabel: string;
  newProjectLabel: string;
  editProjectLabel: string;
  statusLabel: string;
  statusActiveLabel: string;
  statusFrozenLabel: string;
  statusDeletedLabel: string;
  noProjectsLabel: string;
  loginPromptLabel: string;
  loadingLabel: string;
  tabWatchesLabel: string;
  watchListLabel: string;
  unwatchLabel: string;
  noWatchesLabel: string;
  nextBillingMessageLabel?: string;
  nextBillingMessagePlaceholder?: string;
  refundBtnLabel?: string;
  refundConfirmLabel?: string;
  refundPendingLabel?: string;
  refundCompletedLabel?: string;
  refundErrorLabel?: string;
  quickDonateLabel?: string;
  quickDonateConfirmLabel?: string;
  quickDonateSuccessLabel?: string;
  quickDonateErrorLabel?: string;
}

function formatDate(iso: string, locale: Locale): string {
  const d = new Date(iso);
  const loc = locale === "ja" ? "ja-JP" : "en-US";
  return d.toLocaleDateString(loc, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

type TabId = "donations" | "projects" | "watches";

export default function MePage({
  locale,
  tabDonationsLabel,
  tabProjectsLabel,
  donationHistoryLabel,
  recurringDonationsLabel,
  noDonationsLabel,
  noRecurringLabel,
  cancelRecurringLabel,
  cancelledLabel,
  editRecurringLabel,
  pauseRecurringLabel,
  resumeRecurringLabel,
  deleteRecurringLabel,
  pausedLabel,
  intervalMonthlyLabel,
  intervalYearlyLabel,
  intervalLabel,
  editRecurringTitle,
  deleteRecurringConfirmTitle,
  deleteRecurringConfirmLabel,
  saveLabel,
  cancelLabel,
  projectNameLabel,
  amountLabel,
  dateLabel,
  messageLabel,
  newProjectLabel,
  editProjectLabel,
  statusLabel,
  statusActiveLabel,
  statusFrozenLabel,
  statusDeletedLabel,
  noProjectsLabel,
  loginPromptLabel,
  loadingLabel,
  tabWatchesLabel,
  watchListLabel,
  unwatchLabel,
  noWatchesLabel,
  nextBillingMessageLabel,
  nextBillingMessagePlaceholder,
  refundBtnLabel,
  refundConfirmLabel,
  refundPendingLabel,
  refundCompletedLabel,
  refundErrorLabel,
  quickDonateLabel,
  quickDonateConfirmLabel,
  quickDonateSuccessLabel,
  quickDonateErrorLabel,
}: Props) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<TabId>("donations");
  const [donations, setDonations] = useState<Donation[]>([]);
  const [recurring, setRecurring] = useState<RecurringDonation[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [watchedProjects, setWatchedProjects] = useState<Project[]>([]);
  const [updatingStatus, setUpdatingStatus] = useState<string | null>(null);
  const [editingRecurringId, setEditingRecurringId] = useState<string | null>(
    null,
  );
  const [editRecurringAmount, setEditRecurringAmount] = useState(0);
  const [editRecurringInterval, setEditRecurringInterval] = useState<
    "monthly" | "yearly"
  >("monthly");
  const [editRecurringNextMsg, setEditRecurringNextMsg] = useState("");
  const [savingRecurringId, setSavingRecurringId] = useState<string | null>(
    null,
  );
  const [deleteConfirmRecurringId, setDeleteConfirmRecurringId] = useState<
    string | null
  >(null);
  const [refundConfirmId, setRefundConfirmId] = useState<string | null>(null);
  const [refundingId, setRefundingId] = useState<string | null>(null);
  const [refundError, setRefundError] = useState<string | null>(null);
  const [quickDonateConfirmId, setQuickDonateConfirmId] = useState<
    string | null
  >(null);
  const [quickDonatingId, setQuickDonatingId] = useState<string | null>(null);
  const [quickDonateSuccess, setQuickDonateSuccess] = useState<string | null>(
    null,
  );

  const basePath = locale === "en" ? "/en" : "";

  const fetchData = async () => {
    const me = await getMe();
    setUser(me);
    if (me) {
      const [d, r, p, w] = await Promise.all([
        getMyDonations(),
        getMySubscriptions(),
        getMyProjects(),
        getWatchedProjects(),
      ]);
      setDonations(d);
      setRecurring(r);
      setProjects(p);
      setWatchedProjects(w);
    } else {
      setWatchedProjects([]);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, []);

  // モック時: ホスト/メンバー切り替えで再取得（AuthStatus が発火するカスタムイベント）
  useEffect(() => {
    if (typeof window === "undefined") return;
    const handler = () => {
      setLoading(true);
      fetchData();
    };
    window.addEventListener("givers-mock-login-changed", handler);
    return () =>
      window.removeEventListener("givers-mock-login-changed", handler);
  }, []);

  // ウォッチ変更時（プロジェクト詳細でウォッチ/解除したとき）にウォッチ一覧を再取得
  useEffect(() => {
    if (typeof window === "undefined" || !user) return;
    const handler = () => {
      getWatchedProjects()
        .then(setWatchedProjects)
        .catch(() => setWatchedProjects([]));
    };
    window.addEventListener("givers-watch-changed", handler);
    return () => window.removeEventListener("givers-watch-changed", handler);
  }, [user?.id]);

  const handleStartEditRecurring = (r: RecurringDonation) => {
    setEditingRecurringId(r.id);
    setEditRecurringAmount(r.amount);
    setEditRecurringInterval(r.interval ?? "monthly");
    setEditRecurringNextMsg(r.next_billing_message ?? "");
  };

  const handleSaveRecurring = async (id: string) => {
    setSavingRecurringId(id);
    try {
      const current = recurring.find((r) => r.id === id);
      await updateSubscription(id, {
        amount: editRecurringAmount,
        next_billing_message:
          editRecurringNextMsg !== (current?.next_billing_message ?? "")
            ? editRecurringNextMsg
            : undefined,
      });
      setRecurring(await getMySubscriptions());
      setEditingRecurringId(null);
    } finally {
      setSavingRecurringId(null);
    }
  };

  const handlePauseResumeRecurring = async (r: RecurringDonation) => {
    try {
      if (r.status === "paused") {
        await resumeSubscription(r.id);
      } else {
        await pauseSubscription(r.id);
      }
      setRecurring(await getMySubscriptions());
    } catch (e) {
      // ignore
    }
  };

  const handleConfirmDeleteRecurring = async () => {
    const id = deleteConfirmRecurringId;
    if (!id) return;
    setDeleteConfirmRecurringId(null);
    try {
      await deleteSubscription(id);
      setRecurring(await getMySubscriptions());
      if (editingRecurringId === id) setEditingRecurringId(null);
    } catch (e) {
      // ignore
    }
  };

  const handleStatusChange = async (projectId: string, status: string) => {
    setUpdatingStatus(projectId);
    try {
      await updateProject(projectId, { status });
      setProjects(await getMyProjects());
    } finally {
      setUpdatingStatus(null);
    }
  };

  const handleRefund = async (donationId: string) => {
    setRefundConfirmId(null);
    setRefundingId(donationId);
    setRefundError(null);
    try {
      await refundDonationByDonor(donationId);
      setDonations((prev) =>
        prev.map((d) =>
          d.id === donationId ? { ...d, refund_status: "completed" } : d,
        ),
      );
    } catch (e: unknown) {
      setRefundError(refundErrorLabel ?? "Refund failed");
    } finally {
      setRefundingId(null);
    }
  };

  const handleQuickDonate = async (donationId: string) => {
    setQuickDonateConfirmId(null);
    setQuickDonatingId(donationId);
    setQuickDonateSuccess(null);
    try {
      const result = await quickDonate(donationId);
      if (result.status === "succeeded") {
        setQuickDonateSuccess(quickDonateSuccessLabel ?? "Donation completed!");
        setDonations(await getMyDonations());
      } else if (result.status === "requires_action" && result.client_secret) {
        // 3Dセキュア再認証が必要な場合
        // TODO: Stripe.js の confirmCardPayment を呼ぶ
        setQuickDonateSuccess(null);
      }
    } catch {
      setQuickDonateSuccess(null);
      alert(quickDonateErrorLabel ?? "Quick donate failed");
    } finally {
      setQuickDonatingId(null);
    }
  };

  if (loading) {
    return <LoadingSkeleton variant="mePage" />;
  }

  if (!user) {
    return (
      <div className="card" style={{ maxWidth: "28rem", marginTop: "1.5rem" }}>
        <p>{loginPromptLabel}</p>
      </div>
    );
  }

  const formatRecurringAmount = (r: RecurringDonation) => {
    const interval = r.interval ?? "monthly";
    const suffix = interval === "yearly" ? "/年" : "/月";
    return `¥${r.amount.toLocaleString()}${suffix}`;
  };

  return (
    <div style={{ marginTop: "1.5rem" }}>
      <ConfirmDialog
        open={deleteConfirmRecurringId !== null}
        title={deleteRecurringConfirmTitle}
        message={deleteRecurringConfirmLabel}
        confirmLabel={deleteRecurringLabel}
        cancelLabel={cancelLabel}
        danger
        onConfirm={handleConfirmDeleteRecurring}
        onCancel={() => setDeleteConfirmRecurringId(null)}
      />
      {refundBtnLabel && (
        <ConfirmDialog
          open={refundConfirmId !== null}
          title={refundBtnLabel}
          message={refundConfirmLabel ?? ""}
          confirmLabel={refundBtnLabel}
          cancelLabel={cancelLabel}
          danger
          onConfirm={() => {
            if (refundConfirmId) handleRefund(refundConfirmId);
          }}
          onCancel={() => {
            setRefundConfirmId(null);
            setRefundError(null);
          }}
        />
      )}
      {quickDonateLabel && (
        <ConfirmDialog
          open={quickDonateConfirmId !== null}
          title={quickDonateLabel}
          message={(() => {
            const d = donations.find((x) => x.id === quickDonateConfirmId);
            if (!d) return quickDonateConfirmLabel ?? "";
            return (quickDonateConfirmLabel ?? "")
              .replace("{amount}", `¥${d.amount.toLocaleString()}`)
              .replace("{project}", d.project_name);
          })()}
          confirmLabel={quickDonateLabel}
          cancelLabel={cancelLabel}
          onConfirm={() => {
            if (quickDonateConfirmId) handleQuickDonate(quickDonateConfirmId);
          }}
          onCancel={() => setQuickDonateConfirmId(null)}
        />
      )}
      {quickDonateSuccess && (
        <p
          style={{
            margin: "0.75rem 0 0",
            color: "var(--color-success, #22c55e)",
            fontSize: "0.9rem",
          }}
        >
          {quickDonateSuccess}
        </p>
      )}
      {refundError && (
        <p
          style={{
            margin: "0.75rem 0 0",
            color: "var(--color-danger)",
            fontSize: "0.9rem",
          }}
        >
          {refundError}
        </p>
      )}
      {/* タブ */}
      <div
        style={{
          borderBottom: "2px solid var(--color-border)",
          display: "flex",
          gap: "0.5rem",
        }}
      >
        {(["donations", "projects", "watches"] as TabId[]).map((tabId) => (
          <button
            key={tabId}
            type="button"
            onClick={() => setActiveTab(tabId)}
            style={{
              padding: "0.75rem 1.25rem",
              border: "none",
              background: "none",
              cursor: "pointer",
              fontSize: "1rem",
              fontWeight: activeTab === tabId ? 600 : 500,
              color:
                activeTab === tabId
                  ? "var(--color-primary)"
                  : "var(--color-text-muted)",
              borderBottom:
                activeTab === tabId
                  ? "2px solid var(--color-primary)"
                  : "2px solid transparent",
              marginBottom: "-2px",
            }}
          >
            {tabId === "donations" && tabDonationsLabel}
            {tabId === "projects" && tabProjectsLabel}
            {tabId === "watches" && tabWatchesLabel}
          </button>
        ))}
      </div>

      {/* タブコンテンツ */}
      <div style={{ marginTop: "1.5rem" }}>
        {activeTab === "donations" && (
          <>
            <section className="card" style={{ marginBottom: "1.5rem" }}>
              <h2 style={{ marginTop: 0 }}>{donationHistoryLabel}</h2>
              {donations.length === 0 ? (
                <p style={{ color: "var(--color-text-muted)" }}>
                  {noDonationsLabel}
                </p>
              ) : (
                <table style={{ width: "100%", borderCollapse: "collapse" }}>
                  <thead>
                    <tr
                      style={{ borderBottom: "1px solid var(--color-border)" }}
                    >
                      <th
                        style={{ textAlign: "left", padding: "0.5rem 0.75rem" }}
                      >
                        {projectNameLabel}
                      </th>
                      <th
                        style={{
                          textAlign: "right",
                          padding: "0.5rem 0.75rem",
                        }}
                      >
                        {amountLabel}
                      </th>
                      <th
                        style={{ textAlign: "left", padding: "0.5rem 0.75rem" }}
                      >
                        {dateLabel}
                      </th>
                      <th
                        style={{ textAlign: "left", padding: "0.5rem 0.75rem" }}
                      >
                        {messageLabel}
                      </th>
                      {refundBtnLabel && (
                        <th
                          style={{
                            textAlign: "left",
                            padding: "0.5rem 0.75rem",
                          }}
                        />
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {donations.map((d) => (
                      <tr
                        key={d.id}
                        style={{
                          borderBottom: "1px solid var(--color-border)",
                        }}
                      >
                        <td style={{ padding: "0.5rem 0.75rem" }}>
                          <a href={`${basePath}/projects/${d.project_id}`}>
                            {d.project_name}
                          </a>
                        </td>
                        <td
                          style={{
                            textAlign: "right",
                            padding: "0.5rem 0.75rem",
                            whiteSpace: "nowrap",
                          }}
                        >
                          ¥{d.amount.toLocaleString()}
                        </td>
                        <td
                          style={{
                            padding: "0.5rem 0.75rem",
                            whiteSpace: "nowrap",
                          }}
                        >
                          {formatDate(d.created_at, locale)}
                        </td>
                        <td
                          style={{
                            padding: "0.5rem 0.75rem",
                            color: "var(--color-text-muted)",
                          }}
                        >
                          {d.message ?? "—"}
                        </td>
                        <td
                          style={{
                            padding: "0.5rem 0.75rem",
                            whiteSpace: "nowrap",
                          }}
                        >
                          <span
                            style={{
                              display: "flex",
                              gap: "0.25rem",
                              flexWrap: "wrap",
                            }}
                          >
                            {quickDonateLabel && !d.refund_status && (
                              <button
                                type="button"
                                className="btn btn-primary btn-small"
                                style={{
                                  fontSize: "0.8rem",
                                  padding: "0.25rem 0.5rem",
                                }}
                                disabled={quickDonatingId === d.id}
                                onClick={() => setQuickDonateConfirmId(d.id)}
                              >
                                {quickDonatingId === d.id
                                  ? "..."
                                  : quickDonateLabel}
                              </button>
                            )}
                            {refundBtnLabel && (
                              <>
                                {d.refund_status === "completed" ? (
                                  <span
                                    style={{
                                      color: "var(--color-text-muted)",
                                      fontSize: "0.85rem",
                                    }}
                                  >
                                    {refundCompletedLabel}
                                  </span>
                                ) : d.refund_status === "pending" ||
                                  refundingId === d.id ? (
                                  <span
                                    style={{
                                      color: "var(--color-text-muted)",
                                      fontSize: "0.85rem",
                                    }}
                                  >
                                    {refundPendingLabel}
                                  </span>
                                ) : (
                                  <button
                                    type="button"
                                    className="btn btn-small"
                                    style={{
                                      fontSize: "0.8rem",
                                      padding: "0.25rem 0.5rem",
                                    }}
                                    onClick={() => setRefundConfirmId(d.id)}
                                  >
                                    {refundBtnLabel}
                                  </button>
                                )}
                              </>
                            )}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>

            <section className="card">
              <h2 style={{ marginTop: 0 }}>{recurringDonationsLabel}</h2>
              {recurring.length === 0 ? (
                <p style={{ color: "var(--color-text-muted)" }}>
                  {noRecurringLabel}
                </p>
              ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                  {recurring.map((r) => (
                    <li
                      key={r.id}
                      style={{
                        padding: "0.75rem 0",
                        borderBottom: "1px solid var(--color-border)",
                      }}
                    >
                      {editingRecurringId === r.id ? (
                        <div style={{ marginTop: "0.5rem" }}>
                          <h3
                            style={{ margin: "0 0 0.5rem", fontSize: "1rem" }}
                          >
                            {editRecurringTitle}
                          </h3>
                          <div
                            style={{
                              display: "flex",
                              flexWrap: "wrap",
                              gap: "0.5rem",
                              alignItems: "center",
                              marginBottom: "0.5rem",
                            }}
                          >
                            <label
                              style={{
                                display: "flex",
                                alignItems: "center",
                                gap: "0.25rem",
                              }}
                            >
                              <span>{amountLabel}:</span>
                              <input
                                type="number"
                                min={100}
                                step={100}
                                value={editRecurringAmount}
                                onChange={(e) =>
                                  setEditRecurringAmount(
                                    Number(e.target.value) || 0,
                                  )
                                }
                                style={{
                                  width: "6rem",
                                  padding: "0.25rem 0.5rem",
                                }}
                              />
                              <span>円</span>
                            </label>
                            <label
                              style={{
                                display: "flex",
                                alignItems: "center",
                                gap: "0.25rem",
                              }}
                            >
                              <span>{intervalLabel}:</span>
                              <select
                                value={editRecurringInterval}
                                onChange={(e) =>
                                  setEditRecurringInterval(
                                    e.target.value as "monthly" | "yearly",
                                  )
                                }
                                style={{ padding: "0.25rem 0.5rem" }}
                              >
                                <option value="monthly">
                                  {intervalMonthlyLabel}
                                </option>
                                <option value="yearly">
                                  {intervalYearlyLabel}
                                </option>
                              </select>
                            </label>
                          </div>
                          {nextBillingMessageLabel && (
                            <div style={{ marginBottom: "0.5rem" }}>
                              <label
                                style={{
                                  display: "block",
                                  fontSize: "0.85rem",
                                  marginBottom: "0.25rem",
                                }}
                              >
                                {nextBillingMessageLabel}:
                              </label>
                              <textarea
                                value={editRecurringNextMsg}
                                onChange={(e) =>
                                  setEditRecurringNextMsg(e.target.value)
                                }
                                placeholder={nextBillingMessagePlaceholder}
                                rows={2}
                                style={{
                                  width: "100%",
                                  padding: "0.25rem 0.5rem",
                                  border: "1px solid var(--color-border)",
                                  borderRadius: "4px",
                                  fontFamily: "inherit",
                                  fontSize: "0.9rem",
                                }}
                              />
                            </div>
                          )}
                          <div style={{ display: "flex", gap: "0.5rem" }}>
                            <button
                              type="button"
                              className="btn btn-primary"
                              onClick={() => handleSaveRecurring(r.id)}
                              disabled={savingRecurringId === r.id}
                            >
                              {savingRecurringId === r.id ? "..." : saveLabel}
                            </button>
                            <button
                              type="button"
                              className="btn"
                              onClick={() => setEditingRecurringId(null)}
                              disabled={!!savingRecurringId}
                            >
                              {cancelLabel}
                            </button>
                          </div>
                        </div>
                      ) : (
                        <div
                          style={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "space-between",
                            flexWrap: "wrap",
                            gap: "0.5rem",
                          }}
                        >
                          <span>
                            <a href={`${basePath}/projects/${r.project_id}`}>
                              {r.project_name}
                            </a>
                            {" — "}
                            {formatRecurringAmount(r)}
                            {r.status === "cancelled" && (
                              <span
                                style={{
                                  marginLeft: "0.5rem",
                                  color: "var(--color-danger)",
                                }}
                              >
                                ({cancelledLabel})
                              </span>
                            )}
                            {r.status === "paused" && (
                              <span
                                style={{
                                  marginLeft: "0.5rem",
                                  color: "var(--color-warning-muted)",
                                }}
                              >
                                ({pausedLabel})
                              </span>
                            )}
                          </span>
                          {r.status !== "cancelled" && (
                            <span
                              style={{
                                display: "flex",
                                gap: "0.25rem",
                                flexWrap: "wrap",
                              }}
                            >
                              <button
                                type="button"
                                className="btn"
                                style={{ fontSize: "0.8rem" }}
                                onClick={() => handleStartEditRecurring(r)}
                              >
                                {editRecurringLabel}
                              </button>
                              <button
                                type="button"
                                className="btn"
                                style={{ fontSize: "0.8rem" }}
                                onClick={() => handlePauseResumeRecurring(r)}
                              >
                                {r.status === "paused"
                                  ? resumeRecurringLabel
                                  : pauseRecurringLabel}
                              </button>
                              <button
                                type="button"
                                className="btn"
                                style={{ fontSize: "0.8rem" }}
                                onClick={() =>
                                  setDeleteConfirmRecurringId(r.id)
                                }
                              >
                                {deleteRecurringLabel}
                              </button>
                            </span>
                          )}
                        </div>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </>
        )}

        {activeTab === "projects" && (
          <section className="card">
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: "1rem",
              }}
            >
              <h2 style={{ margin: 0 }}>{tabProjectsLabel}</h2>
              <a href={`${basePath}/projects/new`} className="btn btn-primary">
                {newProjectLabel}
              </a>
            </div>
            {projects.length === 0 ? (
              <p style={{ color: "var(--color-text-muted)" }}>
                {noProjectsLabel}
              </p>
            ) : (
              <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                {projects.map((p) => (
                  <li
                    key={p.id}
                    style={{
                      padding: "1rem 0",
                      borderBottom: "1px solid var(--color-border)",
                      display: "flex",
                      flexWrap: "wrap",
                      alignItems: "center",
                      gap: "1rem",
                    }}
                  >
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <a
                        href={`${basePath}/projects/${p.id}`}
                        style={{ fontWeight: 600 }}
                      >
                        {p.name}
                      </a>
                      <span
                        style={{
                          marginLeft: "0.5rem",
                          fontSize: "0.85rem",
                          padding: "0.15rem 0.4rem",
                          borderRadius: "4px",
                          backgroundColor:
                            p.status === "active"
                              ? "var(--color-primary-muted)"
                              : p.status === "frozen"
                                ? "var(--color-warning-muted)"
                                : "var(--color-text-muted)",
                          color:
                            p.status === "deleted"
                              ? "white"
                              : "var(--color-text)",
                        }}
                      >
                        {p.status === "active" && statusActiveLabel}
                        {p.status === "frozen" && statusFrozenLabel}
                        {p.status === "deleted" && statusDeletedLabel}
                        {!["active", "frozen", "deleted"].includes(p.status) &&
                          p.status}
                      </span>
                    </div>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: "0.5rem",
                      }}
                    >
                      <a
                        href={`${basePath}/projects/${p.id}/edit`}
                        className="btn"
                        style={{ fontSize: "0.85rem" }}
                      >
                        {editProjectLabel}
                      </a>
                      <label
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: "0.25rem",
                          fontSize: "0.9rem",
                        }}
                      >
                        <span>{statusLabel}:</span>
                        <select
                          value={p.status}
                          onChange={(e) =>
                            handleStatusChange(p.id, e.target.value)
                          }
                          disabled={updatingStatus === p.id}
                          style={{ padding: "0.25rem 0.5rem" }}
                        >
                          <option value="active">{statusActiveLabel}</option>
                          <option value="frozen">{statusFrozenLabel}</option>
                          <option value="deleted">{statusDeletedLabel}</option>
                        </select>
                      </label>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}

        {activeTab === "watches" && (
          <section className="card">
            <h2 style={{ marginTop: 0 }}>{watchListLabel}</h2>
            {watchedProjects.length === 0 ? (
              <p style={{ color: "var(--color-text-muted)" }}>
                {noWatchesLabel}
              </p>
            ) : (
              <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                {watchedProjects.map((p) => (
                  <li
                    key={p.id}
                    style={{
                      padding: "0.75rem 0",
                      borderBottom: "1px solid var(--color-border)",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "space-between",
                      gap: "0.75rem",
                    }}
                  >
                    <a
                      href={`${basePath}/projects/${p.id}`}
                      style={{ fontWeight: 500 }}
                    >
                      {p.name}
                    </a>
                    <button
                      type="button"
                      className="btn"
                      style={{ fontSize: "0.85rem" }}
                      onClick={async () => {
                        await unwatchProject(p.id);
                        setWatchedProjects((prev) =>
                          prev.filter((x) => x.id !== p.id),
                        );
                      }}
                    >
                      {unwatchLabel}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
