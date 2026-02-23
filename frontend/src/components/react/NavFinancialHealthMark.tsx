import { useEffect, useState } from "react";
import type { Project } from "../../lib/api";
import { getProject, PLATFORM_PROJECT_ID } from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
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

export default function NavFinancialHealthMark({ locale }: Props) {
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    getProject(PLATFORM_PROJECT_ID)
      .then(setProject)
      .catch(() => setError(true));
  }, []);

  if (error || !project) {
    return null;
  }

  const target = monthlyTarget(project);
  const current = project.current_monthly_donations ?? 0;
  const reached = target > 0 && current >= target;
  const rate = target > 0 ? Math.round((current / target) * 100) : 0;

  const titleReached = t(locale, "nav.financialHealthTitleReached");
  const titleNotReached = t(locale, "nav.financialHealthTitleNotReached", {
    rate: String(rate),
  });
  const title = reached ? titleReached : titleNotReached;

  return (
    <span
      className={`nav-financial-health-mark nav-financial-health-mark--${reached ? "reached" : "not-reached"}`}
      role="status"
      aria-label={title}
      title={title}
    >
      {reached ? (
        <>
          <span className="nav-financial-health-mark__icon" aria-hidden="true">
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </span>
          <span className="nav-financial-health-mark__text">
            {t(locale, "nav.financialHealthReached")}
          </span>
        </>
      ) : (
        <>
          <span className="nav-financial-health-mark__icon" aria-hidden="true">
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
          </span>
          <span className="nav-financial-health-mark__text">
            {t(locale, "nav.financialHealthNotReached", { rate: String(rate) })}
          </span>
        </>
      )}
    </span>
  );
}
