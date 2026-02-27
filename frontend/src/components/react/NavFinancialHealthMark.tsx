import { useEffect, useState } from "react";
import { getHostHealth, type PlatformHealthData } from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
}

export default function NavFinancialHealthMark({ locale }: Props) {
  const [health, setHealth] = useState<PlatformHealthData | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    getHostHealth()
      .then(setHealth)
      .catch(() => setError(true));
  }, []);

  if (error || !health) {
    return null;
  }

  const { rate, signal } = health;
  const reached = signal === "green";

  const title = reached
    ? t(locale, "nav.financialHealthTitleReached")
    : t(locale, "nav.financialHealthTitleNotReached", { rate: String(rate) });

  const dotColor =
    signal === "green" ? "#2ecc71" : signal === "yellow" ? "#f1c40f" : "#e74c3c";

  return (
    <span
      className={`nav-financial-health-mark nav-financial-health-mark--${signal}`}
      role="status"
      aria-label={title}
      title={title}
    >
      <span
        className="nav-financial-health-mark__icon"
        aria-hidden="true"
        style={{
          display: "inline-block",
          width: "10px",
          height: "10px",
          borderRadius: "50%",
          backgroundColor: dotColor,
          flexShrink: 0,
        }}
      />
      <span className="nav-financial-health-mark__text">
        {reached
          ? t(locale, "nav.financialHealthReached")
          : t(locale, "nav.financialHealthNotReached", { rate: String(rate) })}
      </span>
    </span>
  );
}
