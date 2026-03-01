import { t, type Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
  message?: string;
  title: string;
  onClose: () => void;
}

export default function ThankYouModal({
  locale,
  message,
  title,
  onClose,
}: Props) {
  const displayMessage =
    message || t(locale, "thankYouMessage.defaultMessage");

  return (
    <>
      <div
        onClick={onClose}
        style={{
          position: "fixed",
          inset: 0,
          background: "rgba(0, 0, 0, 0.5)",
          zIndex: 1100,
        }}
      />
      <div
        role="dialog"
        aria-modal="true"
        style={{
          position: "fixed",
          top: "50%",
          left: "50%",
          transform: "translate(-50%, -50%)",
          background: "var(--color-bg, #fff)",
          borderRadius: "12px",
          padding: "2rem",
          maxWidth: "420px",
          width: "90%",
          zIndex: 1101,
          textAlign: "center",
          boxShadow: "0 8px 32px rgba(0, 0, 0, 0.2)",
        }}
      >
        <div
          style={{
            width: "48px",
            height: "48px",
            borderRadius: "50%",
            background: "var(--color-primary-muted, #a8c9a8)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            margin: "0 auto 1rem",
            fontSize: "1.5rem",
          }}
        >
          &#x2714;
        </div>
        <h2 style={{ margin: "0 0 0.75rem", fontSize: "1.25rem" }}>{title}</h2>
        <p
          style={{
            margin: "0 0 1.5rem",
            color: "var(--color-text-muted, #555)",
            lineHeight: 1.6,
            whiteSpace: "pre-wrap",
          }}
        >
          {displayMessage}
        </p>
        <button
          className="btn btn-primary"
          onClick={onClose}
          style={{ minWidth: "120px" }}
        >
          {t(locale, "thankYouMessage.close")}
        </button>
      </div>
    </>
  );
}
