import { useEffect, useState } from "react";
import type { Project } from "../../lib/api";
import { getProjects, PLATFORM_PROJECT_ID } from "../../lib/api";
import type { Locale } from "../../lib/i18n";
import LoadingSkeleton from "./LoadingSkeleton";

interface Props {
  locale: Locale;
  platformBadge?: string;
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

function achievementRate(project: Project): number {
  const target = monthlyTarget(project);
  const current = project.current_monthly_donations ?? 0;
  return target > 0 ? Math.round((current / target) * 100) : 0;
}

export default function ProjectList({ locale, platformBadge }: Props) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getProjects()
      .then((result) => setProjects(result.projects))
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <LoadingSkeleton variant="projectList" />;
  if (error) return <p style={{ color: "var(--color-danger)" }}>{error}</p>;
  if (projects.length === 0) return <p>プロジェクトはまだありません。</p>;

  return (
    <div className="project-list" style={{ marginTop: "2rem" }}>
      {projects.map((project) => {
        const target = monthlyTarget(project);
        const rate = achievementRate(project);
        const basePath = locale === "en" ? "/en" : "";
        return (
          <a
            key={project.id}
            href={`${basePath}/projects/${project.id}`}
            className="card project-card"
          >
            <div
              className="project-header"
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "flex-start",
                marginBottom: "0.5rem",
              }}
            >
              <div>
                <h2 style={{ margin: 0 }}>
                  {project.name}
                  {project.id === PLATFORM_PROJECT_ID && platformBadge && (
                    <span
                      style={{
                        marginLeft: "0.5rem",
                        fontSize: "0.7rem",
                        fontWeight: 500,
                        color: "var(--color-primary)",
                        backgroundColor: "var(--color-bg-accent)",
                        padding: "0.15rem 0.5rem",
                        borderRadius: "4px",
                      }}
                    >
                      {platformBadge}
                    </span>
                  )}
                </h2>
                {project.owner_name && (
                  <span
                    className="project-owner"
                    style={{
                      display: "block",
                      fontSize: "0.85rem",
                      color: "var(--color-text-muted)",
                    }}
                  >
                    by {project.owner_name}
                  </span>
                )}
              </div>
              {target > 0 && (
                <span
                  className="achievement-badge"
                  data-level={
                    rate >= 80 ? "ok" : rate >= 50 ? "warn" : "danger"
                  }
                >
                  {rate}%
                </span>
              )}
            </div>
            <p
              style={{
                margin: 0,
                color: "var(--color-text-muted)",
                fontSize: "0.95rem",
              }}
            >
              {project.description || ""}
            </p>
            {target > 0 && (
              <p style={{ margin: "0.5rem 0 0", fontSize: "0.9rem" }}>
                月額目標: ¥{target.toLocaleString()}
              </p>
            )}
          </a>
        );
      })}
    </div>
  );
}
