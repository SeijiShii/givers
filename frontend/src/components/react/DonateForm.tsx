import { useState } from "react";
import { t, type Locale } from "../../lib/i18n";
import type { User } from "../../lib/api";

type DonationType = "one_time" | "monthly";

interface Props {
  locale: Locale;
  projectName: string;
  donateLabel: string;
  amountPresets: number[];
  customAmountLabel: string;
  messageLabel: string;
  messagePlaceholder: string;
  submitLabel: string;
  submitLabelMonthly: string;
  thankYouTitle: string;
  thankYouMessageKey: string;
  thankYouMessageMonthlyKey: string;
  oneTimeLabel: string;
  monthlyLabel: string;
  donationTypeLabel?: string;
  /** 利用停止アカウントのときは寄付不可。メッセージを表示する */
  user?: User | null;
  /** 凍結・削除プロジェクトのときは寄付不可。メッセージを表示する */
  projectStatus?: string;
}

export default function DonateForm({
  locale,
  projectName,
  donateLabel,
  amountPresets,
  customAmountLabel,
  messageLabel,
  messagePlaceholder,
  submitLabel,
  submitLabelMonthly,
  thankYouTitle,
  thankYouMessageKey,
  thankYouMessageMonthlyKey,
  oneTimeLabel,
  monthlyLabel,
  donationTypeLabel = "寄付の種類",
  user,
  projectStatus,
}: Props) {
  const [donationType, setDonationType] = useState<DonationType>("one_time");
  const [selectedAmount, setSelectedAmount] = useState<
    number | "custom" | null
  >(null);
  const [customAmount, setCustomAmount] = useState("");
  const [message, setMessage] = useState("");
  const [submitted, setSubmitted] = useState(false);

  if (user?.suspended) {
    return (
      <div
        className="card"
        style={{
          marginTop: "1rem",
          borderColor: "var(--color-danger)",
          background: "var(--color-danger-muted, rgba(200,0,0,0.08))",
        }}
      >
        <p style={{ margin: 0 }}>{t(locale, "errors.accountSuspended")}</p>
      </div>
    );
  }
  if (projectStatus === "frozen") {
    return (
      <div
        className="card"
        style={{ marginTop: "1rem", borderColor: "var(--color-warning)" }}
      >
        <p style={{ margin: 0 }}>{t(locale, "errors.projectFrozen")}</p>
      </div>
    );
  }
  if (projectStatus === "deleted") {
    return (
      <div
        className="card"
        style={{ marginTop: "1rem", borderColor: "var(--color-text-muted)" }}
      >
        <p style={{ margin: 0 }}>{t(locale, "errors.projectDeleted")}</p>
      </div>
    );
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const amount =
      selectedAmount === "custom"
        ? parseInt(customAmount.replace(/\D/g, ""), 10) || 0
        : (selectedAmount ?? 0);
    if (amount <= 0) return;
    setSubmitted(true);
  };

  if (submitted) {
    const amount =
      selectedAmount === "custom"
        ? parseInt(customAmount.replace(/\D/g, ""), 10) || 0
        : (selectedAmount ?? 0);
    const messageKey =
      donationType === "monthly"
        ? thankYouMessageMonthlyKey
        : thankYouMessageKey;
    const isAnonymous = !user;
    return (
      <div
        className="card"
        style={{ marginTop: "1rem", borderColor: "var(--color-primary)" }}
      >
        <h3 style={{ marginTop: 0, color: "var(--color-primary)" }}>
          {thankYouTitle}
        </h3>
        <p style={{ marginBottom: isAnonymous ? "0.75rem" : 0 }}>
          {t(locale, messageKey, {
            amount: `¥${amount.toLocaleString()}`,
            project: projectName,
          })}
        </p>
        {isAnonymous && donationType === "one_time" && (
          <p
            style={{
              margin: 0,
              fontSize: "0.875rem",
              color: "var(--color-text-muted)",
            }}
          >
            {t(locale, "projects.thankYouAnonymousHint")}{" "}
            <a
              href={locale === "en" ? "/en/me" : "/me"}
              style={{ color: "var(--color-primary)" }}
            >
              {t(locale, "projects.thankYouLoginLink")}
            </a>
          </p>
        )}
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} style={{ marginTop: "1rem" }}>
      <h3 style={{ marginTop: 0, marginBottom: "0.75rem" }}>{donateLabel}</h3>
      <div style={{ marginBottom: "1rem" }}>
        <p
          style={{ margin: "0 0 0.5rem", fontSize: "0.95rem", fontWeight: 500 }}
        >
          {donationTypeLabel}
        </p>
        <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.75rem",
              cursor: "pointer",
            }}
          >
            <input
              type="radio"
              name="donationType"
              checked={donationType === "one_time"}
              onChange={() => setDonationType("one_time")}
            />
            <span>{oneTimeLabel}</span>
          </label>
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.75rem",
              cursor: "pointer",
            }}
          >
            <input
              type="radio"
              name="donationType"
              checked={donationType === "monthly"}
              onChange={() => setDonationType("monthly")}
            />
            <span>{monthlyLabel}</span>
          </label>
        </div>
      </div>
      <div style={{ marginBottom: "1rem" }}>
        <p
          style={{ margin: "0 0 0.5rem", fontSize: "0.95rem", fontWeight: 500 }}
        >
          金額を選択
        </p>
        <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
          {amountPresets.map((amount) => (
            <button
              key={amount}
              type="button"
              className={`btn ${selectedAmount === amount ? "btn-primary" : "btn-outline"}`}
              style={{
                ...(selectedAmount !== amount && {
                  backgroundColor: "transparent",
                  border: "1px solid var(--color-primary)",
                  color: "var(--color-primary)",
                }),
              }}
              onClick={() => setSelectedAmount(amount)}
            >
              ¥{amount.toLocaleString()}
            </button>
          ))}
          <button
            type="button"
            className={`btn ${selectedAmount === "custom" ? "btn-primary" : ""}`}
            style={{
              ...(selectedAmount !== "custom" && {
                backgroundColor: "transparent",
                border: "1px solid var(--color-primary)",
                color: "var(--color-primary)",
              }),
            }}
            onClick={() => setSelectedAmount("custom")}
          >
            {customAmountLabel}
          </button>
        </div>
        {selectedAmount === "custom" && (
          <div style={{ marginTop: "0.5rem" }}>
            <input
              type="text"
              placeholder="例: 1500"
              value={customAmount}
              onChange={(e) => setCustomAmount(e.target.value)}
              style={{
                padding: "0.5rem",
                border: "1px solid var(--color-border)",
                borderRadius: "6px",
                width: "120px",
              }}
            />
            <span style={{ marginLeft: "0.5rem", fontSize: "0.9rem" }}>円</span>
          </div>
        )}
      </div>
      <div style={{ marginBottom: "1rem" }}>
        <label
          htmlFor="donate-message"
          style={{
            display: "block",
            marginBottom: "0.25rem",
            fontSize: "0.95rem",
          }}
        >
          {messageLabel}{" "}
          <span
            style={{ color: "var(--color-text-muted)", fontWeight: "normal" }}
          >
            (任意)
          </span>
        </label>
        <textarea
          id="donate-message"
          placeholder={messagePlaceholder}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          rows={3}
          style={{
            width: "100%",
            padding: "0.5rem",
            border: "1px solid var(--color-border)",
            borderRadius: "6px",
            fontFamily: "inherit",
          }}
        />
      </div>
      <button
        type="submit"
        className="btn btn-accent"
        disabled={
          selectedAmount === null ||
          (selectedAmount === "custom" &&
            (!customAmount ||
              parseInt(customAmount.replace(/\D/g, ""), 10) <= 0))
        }
      >
        {donationType === "monthly" ? submitLabelMonthly : submitLabel}
      </button>
    </form>
  );
}
