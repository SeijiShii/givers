import { useState } from "react";
import { t, type Locale } from "../../lib/i18n";
import { createCheckout, type User } from "../../lib/api";

type DonationType = "one_time" | "monthly";

interface Props {
  locale: Locale;
  projectId: string;
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
  projectId,
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
  const isLoggedIn = !!user && !user.suspended;
  const [donationType, setDonationType] = useState<DonationType>("one_time");
  const [selectedAmount, setSelectedAmount] = useState<
    number | "custom" | null
  >(null);
  const [customAmount, setCustomAmount] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const amount =
      selectedAmount === "custom"
        ? parseInt(customAmount.replace(/\D/g, ""), 10) || 0
        : (selectedAmount ?? 0);
    if (amount <= 0) return;

    setSubmitting(true);
    setError(null);
    try {
      const { checkout_url } = await createCheckout({
        project_id: projectId,
        amount,
        currency: "jpy",
        is_recurring: donationType === "monthly",
        message: message || undefined,
        locale: locale === "en" ? "en" : "ja",
      });
      // Stripe Checkout にリダイレクト
      window.location.href = checkout_url;
    } catch (err) {
      setError(err instanceof Error ? err.message : "決済の開始に失敗しました");
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} style={{ marginTop: "1rem" }}>
      <h3 style={{ marginTop: 0, marginBottom: "0.75rem" }}>{donateLabel}</h3>
      {error && (
        <div
          style={{
            padding: "0.5rem 0.75rem",
            marginBottom: "1rem",
            borderRadius: "6px",
            background: "var(--color-danger-muted, rgba(200,0,0,0.08))",
            color: "var(--color-danger, #c00)",
            fontSize: "0.9rem",
          }}
        >
          {error}
        </div>
      )}
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
              cursor: isLoggedIn ? "pointer" : "not-allowed",
              opacity: isLoggedIn ? 1 : 0.5,
            }}
          >
            <input
              type="radio"
              name="donationType"
              checked={donationType === "monthly"}
              onChange={() => setDonationType("monthly")}
              disabled={!isLoggedIn}
            />
            <span>{monthlyLabel}</span>
          </label>
        </div>
        {!isLoggedIn && (
          <p
            style={{
              margin: "0.25rem 0 0",
              fontSize: "0.85rem",
              color: "var(--color-text-muted)",
            }}
          >
            {t(locale, "errors.recurringRequiresLogin")}
          </p>
        )}
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
          submitting ||
          selectedAmount === null ||
          (selectedAmount === "custom" &&
            (!customAmount ||
              parseInt(customAmount.replace(/\D/g, ""), 10) <= 0))
        }
      >
        {submitting
          ? "処理中..."
          : donationType === "monthly"
            ? submitLabelMonthly
            : submitLabel}
      </button>
    </form>
  );
}
