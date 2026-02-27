import { useState, useEffect } from "react";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  url: string;
  title: string;
  locale: Locale;
  shareLabel: string;
  /** DB に保存されたシェアメッセージ（localStorage より優先度低） */
  defaultMessage?: string;
}

const XIcon = () => (
  <svg
    viewBox="0 0 24 24"
    width="16"
    height="16"
    fill="currentColor"
    aria-hidden="true"
  >
    <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
  </svg>
);

const FacebookIcon = () => (
  <svg
    viewBox="0 0 24 24"
    width="16"
    height="16"
    fill="currentColor"
    aria-hidden="true"
  >
    <path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z" />
  </svg>
);

const LineIcon = () => (
  <svg
    viewBox="0 0 24 24"
    width="16"
    height="16"
    fill="currentColor"
    aria-hidden="true"
  >
    <path d="M24 10.304c0-5.369-5.383-9.738-12-9.738S0 4.935 0 10.304c0 4.814 4.27 8.846 10.035 9.608.391.084.922.258 1.057.592.121.303.079.778.039 1.085l-.171 1.027c-.053.303-.242 1.186 1.039.647 1.281-.54 6.911-4.069 9.428-6.967C23.271 14.26 24 12.382 24 10.304zM8.078 12.858H6.012a.535.535 0 01-.534-.535V8.591a.535.535 0 011.068 0v3.197h1.532a.535.535 0 010 1.07zm2.16-.535a.535.535 0 01-1.069 0V8.591a.535.535 0 011.069 0v3.732zm4.918 0a.535.535 0 01-.947.341l-2.09-2.844v2.503a.535.535 0 01-1.069 0V8.591a.535.535 0 01.947-.342l2.09 2.845V8.591a.535.535 0 011.069 0v3.732zm3.87-2.663a.535.535 0 010 1.07h-1.532v.524h1.532a.535.535 0 010 1.069H15.96a.535.535 0 01-.535-.535V8.591c0-.295.24-.535.535-.535h2.066a.535.535 0 010 1.07h-1.532v.534h1.532z" />
  </svg>
);

const CopyIcon = () => (
  <svg
    viewBox="0 0 24 24"
    width="16"
    height="16"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
  >
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
  </svg>
);

type PlatformKey = "X" | "Facebook" | "LINE";

function buildShareUrl(
  platform: PlatformKey,
  pageUrl: string,
  message: string,
): string {
  const encodedUrl = encodeURIComponent(pageUrl);
  const encodedText = encodeURIComponent(message);
  switch (platform) {
    case "X":
      return `https://twitter.com/intent/tweet?text=${encodedText}&url=${encodedUrl}`;
    case "Facebook":
      return `https://www.facebook.com/sharer/sharer.php?u=${encodedUrl}&quote=${encodedText}`;
    case "LINE":
      return `https://social-plugins.line.me/lineit/share?url=${encodedUrl}&text=${encodedText}`;
  }
}

export default function ShareButtons({
  url,
  title,
  locale,
  shareLabel,
  defaultMessage,
}: Props) {
  const storageKey = `givers-share-msg:${url}`;
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedPlatform, setSelectedPlatform] = useState<PlatformKey | null>(
    null,
  );
  const [copyMode, setCopyMode] = useState(false);
  const [copied, setCopied] = useState(false);
  // Priority: localStorage > DB share_message > project title
  const [message, setMessage] = useState(defaultMessage || title);

  useEffect(() => {
    try {
      const saved = localStorage.getItem(storageKey);
      if (saved) setMessage(saved);
    } catch {
      /* localStorage unavailable */
    }
  }, [storageKey]);

  const handleShareClick = (platform: PlatformKey) => {
    setSelectedPlatform(platform);
    setCopyMode(false);
    setCopied(false);
    setDialogOpen(true);
  };

  const handleCopyClick = () => {
    setSelectedPlatform(null);
    setCopyMode(true);
    setCopied(false);
    setDialogOpen(true);
  };

  const handleConfirm = () => {
    try {
      localStorage.setItem(storageKey, message);
    } catch {
      /* ignore */
    }

    if (copyMode) {
      navigator.clipboard
        .writeText(message + "\n" + url)
        .then(() => {
          setCopied(true);
          setTimeout(() => {
            setCopied(false);
            setDialogOpen(false);
          }, 1500);
        })
        .catch(() => {
          setDialogOpen(false);
        });
      return;
    }

    if (!selectedPlatform) return;
    const shareUrl = buildShareUrl(selectedPlatform, url, message);
    window.open(shareUrl, "_blank", "noopener,noreferrer");
    setDialogOpen(false);
  };

  const platforms: { name: PlatformKey; icon: JSX.Element; color: string }[] = [
    { name: "X", icon: <XIcon />, color: "#000000" },
    { name: "Facebook", icon: <FacebookIcon />, color: "#1877F2" },
    { name: "LINE", icon: <LineIcon />, color: "#06C755" },
  ];

  const dialogTitle = copyMode
    ? t(locale, "share.copyLink")
    : selectedPlatform
      ? t(locale, `share.${selectedPlatform.toLowerCase()}`)
      : "";

  return (
    <div className="share-buttons">
      <span className="share-buttons-label">{shareLabel}</span>
      {platforms.map((p) => (
        <button
          key={p.name}
          type="button"
          onClick={() => handleShareClick(p.name)}
          aria-label={t(locale, `share.${p.name.toLowerCase()}`)}
          className="share-btn"
          style={{ backgroundColor: p.color }}
        >
          {p.icon}
          {p.name}
        </button>
      ))}
      <button
        type="button"
        onClick={handleCopyClick}
        aria-label={t(locale, "share.copyLink")}
        className="share-btn"
        style={{ backgroundColor: "#6b7280" }}
      >
        <CopyIcon />
        {t(locale, "share.copyLink")}
      </button>

      {dialogOpen && (
        <div
          className="share-dialog-overlay"
          onClick={() => setDialogOpen(false)}
        >
          <div className="share-dialog" onClick={(e) => e.stopPropagation()}>
            <h3 className="share-dialog-title">{dialogTitle}</h3>
            <textarea
              className="share-dialog-textarea"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              rows={4}
              placeholder={t(locale, "share.messagePlaceholder")}
            />
            <div className="share-dialog-actions">
              <button
                type="button"
                className="btn"
                onClick={() => setDialogOpen(false)}
              >
                {t(locale, "share.cancel")}
              </button>
              <button
                type="button"
                className="btn btn-primary"
                onClick={handleConfirm}
                disabled={copied}
              >
                {copied
                  ? t(locale, "share.linkCopied")
                  : copyMode
                    ? t(locale, "share.copyLink")
                    : t(locale, "share.post")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
