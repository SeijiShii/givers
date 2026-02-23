import { useEffect, useState } from "react";
import ReactMarkdown from "react-markdown";
import type { Project, ProjectUpdate, User } from "../../lib/api";
import {
  getProject,
  getProjectUpdates,
  getMe,
  getRelatedProjects,
  getWatchedProjects,
  watchProject,
  unwatchProject,
  updateProject,
  createProjectUpdate,
  updateProjectUpdate,
  PLATFORM_PROJECT_ID,
} from "../../lib/api";
import DonateForm from "./DonateForm";
import ProjectChart from "./charts/ProjectChart";
import ConfirmDialog from "./ConfirmDialog";
import LoadingSkeleton from "./LoadingSkeleton";
import ShareButtons from "./ShareButtons";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  id: string;
  locale: Locale;
  backLabel: string;
  supportStatus: string;
  supportTitle: string;
  donateLabel: string;
  ownerLabel: string;
  recentSupportersLabel: string;
  anonymousLabel: string;
  donateFormPresets: number[];
  customAmountLabel: string;
  messageLabel: string;
  messagePlaceholder: string;
  thankYouTitle: string;
  donateLabelMonthly: string;
  oneTimeLabel: string;
  monthlyLabel: string;
  donationTypeLabel?: string;
  chartMinAmountLabel: string;
  chartTargetAmountLabel: string;
  chartActualAmountLabel: string;
  chartNoDataLabel: string;
  hostPageLink?: string;
  backHref?: string;
  hideHostPageLink?: boolean;
  tabSupportLabel: string;
  tabOverviewLabel: string;
  tabUpdatesLabel: string;
  updatesEmptyLabel: string;
  editOverviewLabel: string;
  saveLabel: string;
  cancelLabel: string;
  postUpdateLabel: string;
  updateTitlePlaceholder: string;
  updateBodyPlaceholder: string;
  editUpdateLabel: string;
  deleteUpdateLabel: string;
  deleteUpdateConfirmTitle: string;
  deleteUpdateConfirmLabel: string;
  hideUpdateLabel: string;
  hideUpdateConfirmTitle: string;
  hideUpdateConfirmLabel: string;
  showUpdateLabel: string;
  relatedProjectsLabel: string;
  watchLabel: string;
  unwatchLabel: string;
  shareUrl?: string;
  shareLabel: string;
}

function monthlyTarget(project: Project): number {
  if (project.monthly_target != null && project.monthly_target > 0) {
    return project.monthly_target;
  }
  if (project.owner_want_monthly != null && project.owner_want_monthly > 0) {
    return project.owner_want_monthly;
  }
  return 0;
}

function formatUpdateDate(iso: string, locale: Locale): string {
  const d = new Date(iso);
  const now = new Date();
  const diffDays = Math.floor(
    (now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24),
  );
  const loc = locale === "ja" ? "ja-JP" : "en-US";
  if (diffDays === 0)
    return d.toLocaleTimeString(loc, { hour: "2-digit", minute: "2-digit" });
  if (diffDays === 1) return locale === "ja" ? "昨日" : "Yesterday";
  if (diffDays < 7)
    return locale === "ja" ? `${diffDays}日前` : `${diffDays} days ago`;
  return d.toLocaleDateString(loc, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

type TabId = "support" | "updates";

export default function ProjectDetail({
  id,
  locale,
  backLabel,
  supportStatus,
  supportTitle,
  donateLabel,
  ownerLabel,
  recentSupportersLabel,
  anonymousLabel,
  donateFormPresets,
  customAmountLabel,
  messageLabel,
  messagePlaceholder,
  thankYouTitle,
  donateLabelMonthly,
  oneTimeLabel,
  monthlyLabel,
  donationTypeLabel,
  chartMinAmountLabel,
  chartTargetAmountLabel,
  chartActualAmountLabel,
  chartNoDataLabel,
  hostPageLink,
  backHref,
  hideHostPageLink,
  tabSupportLabel,
  tabOverviewLabel,
  tabUpdatesLabel,
  updatesEmptyLabel,
  editOverviewLabel,
  saveLabel,
  cancelLabel,
  postUpdateLabel,
  updateTitlePlaceholder,
  updateBodyPlaceholder,
  editUpdateLabel,
  deleteUpdateLabel,
  deleteUpdateConfirmTitle,
  deleteUpdateConfirmLabel,
  hideUpdateLabel,
  hideUpdateConfirmTitle,
  hideUpdateConfirmLabel,
  showUpdateLabel,
  relatedProjectsLabel,
  watchLabel,
  unwatchLabel,
  shareUrl,
  shareLabel,
}: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [updates, setUpdates] = useState<ProjectUpdate[]>([]);
  const [relatedProjects, setRelatedProjects] = useState<Project[]>([]);
  const [isWatching, setIsWatching] = useState(false);
  const [loadingWatch, setLoadingWatch] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<TabId>("support");
  const [me, setMe] = useState<User | null>(null);
  const [editingOverview, setEditingOverview] = useState(false);
  const [overviewDraft, setOverviewDraft] = useState("");
  const [savingOverview, setSavingOverview] = useState(false);
  const [updateTitle, setUpdateTitle] = useState("");
  const [updateBody, setUpdateBody] = useState("");
  const [postingUpdate, setPostingUpdate] = useState(false);
  const [editingUpdateId, setEditingUpdateId] = useState<string | null>(null);
  const [editUpdateTitle, setEditUpdateTitle] = useState("");
  const [editUpdateBody, setEditUpdateBody] = useState("");
  const [savingUpdateId, setSavingUpdateId] = useState<string | null>(null);
  const [deletingUpdateId, setDeletingUpdateId] = useState<string | null>(null);
  const [showingUpdateId, setShowingUpdateId] = useState<string | null>(null);
  const [deleteConfirmUpdateId, setDeleteConfirmUpdateId] = useState<
    string | null
  >(null);

  const isOwner = me && project && project.owner_id === me.id;

  useEffect(() => {
    getProject(id)
      .then(setProject)
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    getMe()
      .then((u) => setMe(u ?? null))
      .catch(() => setMe(null));
  }, []);

  useEffect(() => {
    if (id) {
      getProjectUpdates(id)
        .then(setUpdates)
        .catch(() => setUpdates([]));
    }
  }, [id]);

  useEffect(() => {
    if (project?.id) {
      getRelatedProjects(project.id, 4)
        .then(setRelatedProjects)
        .catch(() => setRelatedProjects([]));
    } else {
      setRelatedProjects([]);
    }
  }, [project?.id]);

  useEffect(() => {
    if (me && project?.id) {
      getWatchedProjects()
        .then((list) => setIsWatching(list.some((p) => p.id === project.id)))
        .catch(() => setIsWatching(false));
    } else {
      setIsWatching(false);
    }
  }, [me?.id, project?.id]);

  useEffect(() => {
    if (project && !editingOverview) {
      setOverviewDraft(project.overview ?? project.description ?? "");
    }
  }, [project, editingOverview]);

  if (loading) return <LoadingSkeleton variant="projectDetail" />;
  if (error) return <p style={{ color: "var(--color-danger)" }}>{error}</p>;
  if (!project) return null;

  const target = monthlyTarget(project);
  const currentMonthly = project.current_monthly_donations ?? 0;
  const achievementRate =
    target > 0 ? Math.round((currentMonthly / target) * 100) : 0;

  const basePath = locale === "en" ? "/en" : "";
  const backUrl = backHref ?? `${basePath}/projects`;

  const overview = project.overview ?? project.description ?? "";

  const handleSaveOverview = async () => {
    if (!project) return;
    setSavingOverview(true);
    try {
      const updated = await updateProject(project.id, {
        overview: overviewDraft,
      });
      setProject(updated);
      setEditingOverview(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSavingOverview(false);
    }
  };

  const handlePostUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!updateBody.trim()) return;
    setPostingUpdate(true);
    try {
      const newUpdate = await createProjectUpdate(id, {
        title: updateTitle.trim() || null,
        body: updateBody.trim(),
      });
      setUpdates((prev) => [newUpdate, ...prev]);
      setUpdateTitle("");
      setUpdateBody("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to post");
    } finally {
      setPostingUpdate(false);
    }
  };

  const handleStartEditUpdate = (u: ProjectUpdate) => {
    setEditingUpdateId(u.id);
    setEditUpdateTitle(u.title ?? "");
    setEditUpdateBody(u.body);
  };

  const handleCancelEditUpdate = () => {
    setEditingUpdateId(null);
    setEditUpdateTitle("");
    setEditUpdateBody("");
  };

  const handleSaveUpdate = async (updateId: string) => {
    setSavingUpdateId(updateId);
    try {
      const updated = await updateProjectUpdate(id, updateId, {
        title: editUpdateTitle.trim() || null,
        body: editUpdateBody.trim(),
      });
      setUpdates((prev) => prev.map((u) => (u.id === updateId ? updated : u)));
      handleCancelEditUpdate();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSavingUpdateId(null);
    }
  };

  const handleRequestDeleteUpdate = (updateId: string) => {
    setDeleteConfirmUpdateId(updateId);
  };

  const handleConfirmHideUpdate = async () => {
    const updateId = deleteConfirmUpdateId;
    if (!updateId) return;
    setDeleteConfirmUpdateId(null);
    setDeletingUpdateId(updateId);
    try {
      await updateProjectUpdate(id, updateId, { visible: false });
      setUpdates((prev) =>
        prev.map((u) => (u.id === updateId ? { ...u, visible: false } : u)),
      );
      if (editingUpdateId === updateId) handleCancelEditUpdate();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to hide");
    } finally {
      setDeletingUpdateId(null);
    }
  };

  const handleShowUpdate = async (updateId: string) => {
    setShowingUpdateId(updateId);
    try {
      await updateProjectUpdate(id, updateId, { visible: true });
      setUpdates((prev) =>
        prev.map((u) => (u.id === updateId ? { ...u, visible: true } : u)),
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to show");
    } finally {
      setShowingUpdateId(null);
    }
  };

  return (
    <div className="project-detail">
      <ConfirmDialog
        open={deleteConfirmUpdateId !== null}
        title={hideUpdateConfirmTitle}
        message={hideUpdateConfirmLabel}
        confirmLabel={hideUpdateLabel}
        cancelLabel={cancelLabel}
        danger
        onConfirm={handleConfirmHideUpdate}
        onCancel={() => setDeleteConfirmUpdateId(null)}
      />
      <a
        href={backUrl}
        style={{
          color: "var(--color-primary)",
          textDecoration: "none",
          fontSize: "0.9rem",
        }}
      >
        ← {backLabel}
      </a>

      {/* ヒーロー画像 */}
      <div
        className="project-hero"
        style={{
          marginTop: "1rem",
          marginBottom: "1.5rem",
          borderRadius: "12px",
          overflow: "hidden",
          backgroundColor: "var(--color-bg-subtle)",
          aspectRatio: "800 / 400",
          maxHeight: "400px",
        }}
      >
        {project.image_url ? (
          <img
            src={project.image_url}
            alt=""
            style={{
              width: "100%",
              height: "100%",
              objectFit: "cover",
              display: "block",
            }}
          />
        ) : (
          <div
            style={{
              width: "100%",
              height: "100%",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: "4rem",
              color: "var(--color-primary-muted)",
            }}
          >
            📦
          </div>
        )}
      </div>

      <h1 style={{ marginTop: 0 }}>
        {project.name}
        {project.id === PLATFORM_PROJECT_ID &&
          hostPageLink &&
          !hideHostPageLink && (
            <a
              href={`${basePath}/host`}
              style={{
                marginLeft: "0.5rem",
                fontSize: "0.75rem",
                fontWeight: 500,
                color: "var(--color-primary)",
              }}
            >
              ({hostPageLink})
            </a>
          )}
      </h1>
      {project.owner_name && (
        <p
          style={{
            marginTop: "0.25rem",
            color: "var(--color-text-muted)",
            fontSize: "0.95rem",
          }}
        >
          {ownerLabel}: {project.owner_name}
        </p>
      )}
      {me && project.owner_id !== me.id ? (
        <p style={{ marginTop: "0.5rem" }}>
          <button
            type="button"
            className="btn"
            disabled={loadingWatch}
            onClick={async () => {
              if (!project) return;
              setLoadingWatch(true);
              try {
                if (isWatching) {
                  await unwatchProject(project.id);
                  setIsWatching(false);
                } else {
                  await watchProject(project.id);
                  setIsWatching(true);
                }
                if (typeof window !== "undefined") {
                  window.dispatchEvent(new CustomEvent("givers-watch-changed"));
                }
              } finally {
                setLoadingWatch(false);
              }
            }}
          >
            {loadingWatch ? "..." : isWatching ? unwatchLabel : watchLabel}
          </button>
        </p>
      ) : null}
      <p style={{ marginTop: "0.5rem", color: "var(--color-text-muted)" }}>
        {project.description || ""}
      </p>

      {shareUrl && (
        <ShareButtons
          url={shareUrl}
          title={project.name}
          locale={locale}
          shareLabel={shareLabel}
          defaultMessage={project.share_message}
        />
      )}

      {project.owner_want_monthly != null && project.owner_want_monthly > 0 && (
        <p style={{ marginTop: "1rem", fontWeight: 600 }}>
          最低希望額: 月額 ¥{project.owner_want_monthly.toLocaleString()}
        </p>
      )}
      {project.cost_items && project.cost_items.length > 0 && (
        <div style={{ marginTop: "0.5rem", color: "var(--color-text-muted)" }}>
          <p style={{ marginBottom: "0.25rem" }}>
            必要額（コスト内訳）: 月額 ¥
            {(project.monthly_target ?? 0).toLocaleString()}
          </p>
          <ul style={{ margin: 0, paddingLeft: "1.2rem", fontSize: "0.9rem" }}>
            {project.cost_items.map((ci, i) => (
              <li key={ci.id ?? i}>
                {ci.label}: ¥
                {ci.unit_type === "daily_x_days"
                  ? (ci.rate_per_day * ci.days_per_month).toLocaleString()
                  : ci.amount_monthly.toLocaleString()}
                /月
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* 概要（常時表示・寄付者をモチベートする大切な情報） */}
      <section
        className="project-overview-always"
        style={{ marginTop: "2rem" }}
        aria-label={tabOverviewLabel}
      >
        <h2
          style={{
            fontSize: "1.15rem",
            marginTop: 0,
            marginBottom: "0.75rem",
            color: "var(--color-primary)",
          }}
        >
          {tabOverviewLabel}
        </h2>
        <div
          className="card project-overview-markdown"
          style={{ padding: "1.5rem" }}
        >
          {isOwner && !editingOverview && (
            <div style={{ marginBottom: "1rem" }}>
              <button
                type="button"
                className="btn"
                onClick={() => setEditingOverview(true)}
              >
                {editOverviewLabel}
              </button>
            </div>
          )}
          {isOwner && editingOverview ? (
            <div style={{ maxWidth: "65ch" }}>
              <textarea
                value={overviewDraft}
                onChange={(e) => setOverviewDraft(e.target.value)}
                rows={16}
                style={{
                  width: "100%",
                  padding: "0.75rem",
                  fontFamily: "inherit",
                  fontSize: "0.95rem",
                  lineHeight: 1.6,
                }}
                placeholder="Markdown で概要を記述"
              />
              <div
                style={{ marginTop: "1rem", display: "flex", gap: "0.5rem" }}
              >
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={handleSaveOverview}
                  disabled={savingOverview}
                >
                  {savingOverview ? "..." : saveLabel}
                </button>
                <button
                  type="button"
                  className="btn"
                  onClick={() => {
                    setEditingOverview(false);
                    setOverviewDraft(overview);
                  }}
                  disabled={savingOverview}
                >
                  {cancelLabel}
                </button>
              </div>
            </div>
          ) : (
            <div style={{ maxWidth: "65ch" }}>
              <ReactMarkdown
                components={{
                  h1: ({ children }) => (
                    <h1
                      style={{
                        marginTop: "1.5rem",
                        marginBottom: "0.5rem",
                        fontSize: "1.4rem",
                      }}
                    >
                      {children}
                    </h1>
                  ),
                  h2: ({ children }) => (
                    <h2
                      style={{
                        marginTop: "1.5rem",
                        marginBottom: "0.5rem",
                        fontSize: "1.2rem",
                      }}
                    >
                      {children}
                    </h2>
                  ),
                  h3: ({ children }) => (
                    <h3
                      style={{
                        marginTop: "1.25rem",
                        marginBottom: "0.5rem",
                        fontSize: "1.1rem",
                      }}
                    >
                      {children}
                    </h3>
                  ),
                  p: ({ children }) => (
                    <p style={{ margin: "0.5rem 0", lineHeight: 1.7 }}>
                      {children}
                    </p>
                  ),
                  ul: ({ children }) => (
                    <ul style={{ margin: "0.5rem 0", paddingLeft: "1.5rem" }}>
                      {children}
                    </ul>
                  ),
                  ol: ({ children }) => (
                    <ol style={{ margin: "0.5rem 0", paddingLeft: "1.5rem" }}>
                      {children}
                    </ol>
                  ),
                  li: ({ children }) => (
                    <li style={{ margin: "0.25rem 0" }}>{children}</li>
                  ),
                  a: ({ href, children }) => (
                    <a
                      href={href}
                      style={{
                        color: "var(--color-primary)",
                        textDecoration: "underline",
                      }}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {children}
                    </a>
                  ),
                  strong: ({ children }) => (
                    <strong style={{ fontWeight: 600 }}>{children}</strong>
                  ),
                  pre: ({ children }) => (
                    <pre
                      style={{
                        backgroundColor: "var(--color-bg-subtle)",
                        padding: "1rem",
                        borderRadius: "8px",
                        overflow: "auto",
                        fontSize: "0.9em",
                      }}
                    >
                      {children}
                    </pre>
                  ),
                  code: ({ children }) => (
                    <code
                      style={{
                        backgroundColor: "var(--color-bg-subtle)",
                        padding: "0.1rem 0.3rem",
                        borderRadius: "4px",
                        fontSize: "0.9em",
                      }}
                    >
                      {children}
                    </code>
                  ),
                }}
              >
                {overview}
              </ReactMarkdown>
            </div>
          )}
        </div>
      </section>

      {/* タブ（支援状況・アップデート） */}
      <div
        className="project-tabs"
        style={{
          marginTop: "2rem",
          borderBottom: "2px solid var(--color-border)",
          display: "flex",
          gap: "0.5rem",
        }}
      >
        {(["support", "updates"] as TabId[]).map((tabId) => (
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
            {tabId === "support" && tabSupportLabel}
            {tabId === "updates" && tabUpdatesLabel}
          </button>
        ))}
      </div>

      {/* タブコンテンツ */}
      <div className="project-tab-content" style={{ marginTop: "1.5rem" }}>
        {activeTab === "support" && (
          <>
            <div className="card" style={{ marginBottom: "1.5rem" }}>
              <h2 style={{ marginTop: 0 }}>{supportStatus}</h2>
              <div className="achievement-bar" style={{ margin: "1rem 0" }}>
                <div
                  className="achievement-fill"
                  style={{ width: `${Math.min(achievementRate, 100)}%` }}
                />
              </div>
              <p>
                {t(locale, "projects.supportStatusDetail", {
                  target: target.toLocaleString(),
                  current: currentMonthly.toLocaleString(),
                  rate: String(achievementRate),
                })}
              </p>
              <ProjectChart
                projectId={project.id}
                minAmountLabel={chartMinAmountLabel}
                targetAmountLabel={chartTargetAmountLabel}
                actualAmountLabel={chartActualAmountLabel}
                noDataLabel={chartNoDataLabel}
              />
            </div>

            {project.recent_supporters &&
              project.recent_supporters.length > 0 && (
                <div className="card" style={{ marginBottom: "1.5rem" }}>
                  <h2 style={{ fontSize: "1.1rem", marginTop: 0 }}>
                    {recentSupportersLabel}
                  </h2>
                  <ul
                    style={{
                      margin: "0.5rem 0 0",
                      paddingLeft: "1.25rem",
                      fontSize: "0.95rem",
                    }}
                  >
                    {project.recent_supporters.slice(0, 5).map((s, i) => (
                      <li key={i}>
                        {s.name ?? anonymousLabel}: ¥{s.amount.toLocaleString()}
                      </li>
                    ))}
                  </ul>
                </div>
              )}

            <div className="card accent-line">
              <h2 style={{ marginTop: 0 }}>{supportTitle}</h2>
              <DonateForm
                locale={locale}
                projectId={project.id}
                projectName={project.name}
                donateLabel={donateLabel}
                amountPresets={donateFormPresets}
                customAmountLabel={customAmountLabel}
                messageLabel={messageLabel}
                messagePlaceholder={messagePlaceholder}
                submitLabel={donateLabel}
                submitLabelMonthly={donateLabelMonthly}
                thankYouTitle={thankYouTitle}
                thankYouMessageKey="projects.thankYouMessage"
                thankYouMessageMonthlyKey="projects.thankYouMessageMonthly"
                oneTimeLabel={oneTimeLabel}
                monthlyLabel={monthlyLabel}
                donationTypeLabel={donationTypeLabel}
                user={me}
                projectStatus={project.status}
              />
            </div>
          </>
        )}

        {activeTab === "updates" && (
          <div className="card" style={{ padding: "1.5rem" }}>
            {isOwner && (
              <form
                onSubmit={handlePostUpdate}
                style={{
                  marginBottom: "1.5rem",
                  paddingBottom: "1.5rem",
                  borderBottom: "1px solid var(--color-border)",
                }}
              >
                <h3 style={{ marginTop: 0, marginBottom: "1rem" }}>
                  {postUpdateLabel}
                </h3>
                <input
                  type="text"
                  value={updateTitle}
                  onChange={(e) => setUpdateTitle(e.target.value)}
                  placeholder={updateTitlePlaceholder}
                  style={{
                    width: "100%",
                    padding: "0.5rem",
                    marginBottom: "0.5rem",
                  }}
                />
                <textarea
                  value={updateBody}
                  onChange={(e) => setUpdateBody(e.target.value)}
                  placeholder={updateBodyPlaceholder}
                  rows={4}
                  required
                  style={{
                    width: "100%",
                    padding: "0.5rem",
                    marginBottom: "0.5rem",
                    fontFamily: "inherit",
                  }}
                />
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={postingUpdate || !updateBody.trim()}
                >
                  {postingUpdate ? "..." : postUpdateLabel}
                </button>
              </form>
            )}
            {(() => {
              const visibleUpdates = isOwner
                ? updates
                : updates.filter((u) => u.visible !== false);
              if (visibleUpdates.length === 0) {
                return (
                  <p style={{ color: "var(--color-text-muted)" }}>
                    {updatesEmptyLabel}
                  </p>
                );
              }
              return (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                  {visibleUpdates.map((u) => {
                    const isHidden = u.visible === false;
                    return (
                      <li
                        key={u.id}
                        style={{
                          padding: "1rem 0",
                          borderBottom: "1px solid var(--color-border-light)",
                          ...(isHidden
                            ? {
                                opacity: 0.7,
                                backgroundColor: "var(--color-bg-subtle)",
                                padding: "1rem",
                                borderRadius: "8px",
                              }
                            : {}),
                        }}
                      >
                        <div
                          style={{
                            fontSize: "0.85rem",
                            color: "var(--color-text-muted)",
                            marginBottom: "0.25rem",
                            display: "flex",
                            justifyContent: "space-between",
                            alignItems: "center",
                            flexWrap: "wrap",
                            gap: "0.5rem",
                          }}
                        >
                          <span>
                            {u.author_name && <span>{u.author_name}</span>}
                            <span style={{ marginLeft: "0.5rem" }}>
                              {formatUpdateDate(u.created_at, locale)}
                            </span>
                            {isHidden && (
                              <span
                                style={{
                                  marginLeft: "0.5rem",
                                  fontStyle: "italic",
                                }}
                              >
                                ({hideUpdateLabel})
                              </span>
                            )}
                          </span>
                          {isOwner && editingUpdateId !== u.id && (
                            <span style={{ display: "flex", gap: "0.25rem" }}>
                              {isHidden ? (
                                <button
                                  type="button"
                                  className="btn btn-primary"
                                  style={{
                                    fontSize: "0.75rem",
                                    padding: "0.2rem 0.4rem",
                                  }}
                                  onClick={() => handleShowUpdate(u.id)}
                                  disabled={!!showingUpdateId}
                                >
                                  {showingUpdateId === u.id
                                    ? "..."
                                    : showUpdateLabel}
                                </button>
                              ) : (
                                <>
                                  <button
                                    type="button"
                                    className="btn"
                                    style={{
                                      fontSize: "0.75rem",
                                      padding: "0.2rem 0.4rem",
                                    }}
                                    onClick={() => handleStartEditUpdate(u)}
                                    disabled={!!deletingUpdateId}
                                  >
                                    {editUpdateLabel}
                                  </button>
                                  <button
                                    type="button"
                                    className="btn"
                                    style={{
                                      fontSize: "0.75rem",
                                      padding: "0.2rem 0.4rem",
                                    }}
                                    onClick={() =>
                                      handleRequestDeleteUpdate(u.id)
                                    }
                                    disabled={!!deletingUpdateId}
                                  >
                                    {deletingUpdateId === u.id
                                      ? "..."
                                      : hideUpdateLabel}
                                  </button>
                                </>
                              )}
                            </span>
                          )}
                        </div>
                        {editingUpdateId === u.id ? (
                          <div style={{ marginTop: "0.5rem" }}>
                            <input
                              type="text"
                              value={editUpdateTitle}
                              onChange={(e) =>
                                setEditUpdateTitle(e.target.value)
                              }
                              placeholder={updateTitlePlaceholder}
                              style={{
                                width: "100%",
                                padding: "0.5rem",
                                marginBottom: "0.5rem",
                              }}
                            />
                            <textarea
                              value={editUpdateBody}
                              onChange={(e) =>
                                setEditUpdateBody(e.target.value)
                              }
                              placeholder={updateBodyPlaceholder}
                              rows={4}
                              style={{
                                width: "100%",
                                padding: "0.5rem",
                                marginBottom: "0.5rem",
                                fontFamily: "inherit",
                              }}
                            />
                            <div style={{ display: "flex", gap: "0.5rem" }}>
                              <button
                                type="button"
                                className="btn btn-primary"
                                style={{ fontSize: "0.85rem" }}
                                onClick={() => handleSaveUpdate(u.id)}
                                disabled={savingUpdateId === u.id}
                              >
                                {savingUpdateId === u.id ? "..." : saveLabel}
                              </button>
                              <button
                                type="button"
                                className="btn"
                                style={{ fontSize: "0.85rem" }}
                                onClick={handleCancelEditUpdate}
                                disabled={!!savingUpdateId}
                              >
                                {cancelLabel}
                              </button>
                            </div>
                          </div>
                        ) : (
                          <>
                            {u.title && (
                              <h3
                                style={{
                                  margin: "0.25rem 0",
                                  fontSize: "1rem",
                                }}
                              >
                                {u.title}
                              </h3>
                            )}
                            <p
                              style={{
                                margin: 0,
                                whiteSpace: "pre-wrap",
                                lineHeight: 1.6,
                              }}
                            >
                              {u.body}
                            </p>
                          </>
                        )}
                      </li>
                    );
                  })}
                </ul>
              );
            })()}
          </div>
        )}
      </div>

      {relatedProjects.length > 0 && (
        <div className="card" style={{ marginTop: "2rem" }}>
          <h2 style={{ fontSize: "1.1rem", marginTop: 0 }}>
            {relatedProjectsLabel}
          </h2>
          <ul
            style={{
              margin: "0.75rem 0 0",
              paddingLeft: 0,
              listStyle: "none",
              display: "flex",
              flexDirection: "column",
              gap: "0.5rem",
            }}
          >
            {relatedProjects.map((p) => {
              const targetVal = monthlyTarget(p);
              const rate =
                targetVal > 0
                  ? Math.round(
                      ((p.current_monthly_donations ?? 0) / targetVal) * 100,
                    )
                  : 0;
              return (
                <li key={p.id}>
                  <a
                    href={`${basePath}/projects/${p.id}`}
                    style={{
                      display: "block",
                      padding: "0.5rem 0",
                      color: "var(--color-primary)",
                      textDecoration: "none",
                      fontWeight: 500,
                      borderBottom: "1px solid var(--color-border-light)",
                    }}
                  >
                    {p.name}
                    {targetVal > 0 && (
                      <span
                        style={{
                          marginLeft: "0.5rem",
                          fontSize: "0.85rem",
                          color: "var(--color-text-muted)",
                          fontWeight: 400,
                        }}
                      >
                        — {rate}%
                      </span>
                    )}
                  </a>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}
