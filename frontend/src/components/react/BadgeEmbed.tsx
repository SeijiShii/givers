import { useState } from "react";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  projectId: string;
  projectUrl: string;
  badgeUrl: string;
  locale: Locale;
  isOwner?: boolean;
}

export default function BadgeEmbed({
  projectId,
  projectUrl,
  badgeUrl,
  locale,
  isOwner,
}: Props) {
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const markdownCode = `[![GIVErS](${badgeUrl})](${projectUrl})`;
  const htmlCode = `<a href="${projectUrl}"><img src="${badgeUrl}" alt="GIVErS"></a>`;

  const handleCopy = (text: string, field: string) => {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopiedField(field);
        setTimeout(() => setCopiedField(null), 2000);
      })
      .catch(() => {});
  };

  const inputStyle: React.CSSProperties = {
    flex: 1,
    padding: "0.4rem 0.6rem",
    fontSize: "0.8rem",
    fontFamily: "monospace",
    border: "1px solid var(--color-border, #ddd)",
    borderRadius: "4px",
    background: "var(--color-bg-accent, #f9f9f9)",
    color: "var(--color-text, #333)",
    minWidth: 0,
  };

  const copyBtnStyle: React.CSSProperties = {
    padding: "0.4rem 0.75rem",
    fontSize: "0.8rem",
    whiteSpace: "nowrap",
  };

  // 閲覧者にはバッジ画像のみ表示
  if (!isOwner) {
    return (
      <div className="badge-embed" style={{ marginTop: "1rem" }}>
        <img
          src={badgeUrl}
          alt="GIVErS badge"
          style={{ height: 20, display: "block" }}
        />
      </div>
    );
  }

  // オーナーにはバッジ + 埋め込みコードを表示
  return (
    <div
      className="badge-embed"
      style={{
        marginTop: "1rem",
        padding: "1rem",
        border: "1px solid var(--color-border, #ddd)",
        borderRadius: "8px",
        background: "var(--color-bg-accent, #f9f9f9)",
      }}
    >
      <h4 style={{ margin: "0 0 0.25rem", fontSize: "0.95rem" }}>
        {t(locale, "badge.title")}
      </h4>
      <p
        style={{
          margin: "0 0 0.75rem",
          fontSize: "0.8rem",
          color: "var(--color-text-muted, #666)",
        }}
      >
        {t(locale, "badge.description")}
      </p>

      <div style={{ marginBottom: "0.75rem" }}>
        <img
          src={badgeUrl}
          alt="GIVErS badge"
          style={{ height: 20, display: "block" }}
        />
      </div>

      <div style={{ marginBottom: "0.5rem" }}>
        <label
          style={{
            display: "block",
            fontWeight: 600,
            fontSize: "0.8rem",
            marginBottom: "0.25rem",
          }}
        >
          Markdown
        </label>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <input type="text" readOnly value={markdownCode} style={inputStyle} />
          <button
            type="button"
            className="btn"
            style={copyBtnStyle}
            onClick={() => handleCopy(markdownCode, "md")}
          >
            {copiedField === "md"
              ? t(locale, "badge.copied")
              : t(locale, "badge.copy")}
          </button>
        </div>
      </div>

      <div>
        <label
          style={{
            display: "block",
            fontWeight: 600,
            fontSize: "0.8rem",
            marginBottom: "0.25rem",
          }}
        >
          HTML
        </label>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <input type="text" readOnly value={htmlCode} style={inputStyle} />
          <button
            type="button"
            className="btn"
            style={copyBtnStyle}
            onClick={() => handleCopy(htmlCode, "html")}
          >
            {copiedField === "html"
              ? t(locale, "badge.copied")
              : t(locale, "badge.copy")}
          </button>
        </div>
      </div>
    </div>
  );
}
