import { useState, useEffect } from "react";
import {
  getAuthProviders,
  getGoogleLoginUrl,
  getGitHubLoginUrl,
  getDiscordLoginUrl,
  getAppleLoginUrl,
  getMe,
  MOCK_LOGIN_MODE_KEY,
} from "../../lib/api";
import GoogleLogoSvg from "./GoogleLogoSvg";

const MOCK_MODE = import.meta.env.PUBLIC_MOCK_MODE === "true";

interface Props {
  open: boolean;
  locale: string;
  onClose: () => void;
}

export default function LoginDialog({ open, locale, onClose }: Props) {
  const [providers, setProviders] = useState<string[]>(["google"]);
  const [showEmailForm, setShowEmailForm] = useState(false);
  const [emailValue, setEmailValue] = useState("");
  const [emailSubmitted, setEmailSubmitted] = useState(false);

  useEffect(() => {
    if (open) {
      getAuthProviders()
        .then((data) => setProviders(data.providers))
        .catch(() => setProviders(["google"]));
      document.body.style.overflow = "hidden";
      setShowEmailForm(false);
      setEmailValue("");
      setEmailSubmitted(false);
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, onClose]);

  if (!open) return null;

  const handleLogin = async (provider: string) => {
    try {
      let url: string;
      switch (provider) {
        case "google": {
          const res = await getGoogleLoginUrl();
          url = res.url;
          break;
        }
        case "github": {
          const res = await getGitHubLoginUrl();
          url = res.url;
          break;
        }
        case "discord": {
          const res = await getDiscordLoginUrl();
          url = res.url;
          break;
        }
        case "apple": {
          const res = await getAppleLoginUrl();
          url = res.url;
          break;
        }
        default:
          return;
      }
      window.location.href = url;
    } catch (err) {
      console.error(`[LoginDialog] ${provider} login failed:`, err);
    }
  };

  const handleMockLogin = () => {
    if (typeof window !== "undefined" && window.localStorage) {
      window.localStorage.setItem(MOCK_LOGIN_MODE_KEY, "host");
      getMe()
        .then(() => {
          window.dispatchEvent(new CustomEvent("givers-mock-login-changed"));
          window.location.reload();
        })
        .catch(() => {});
    }
  };

  const handleEmailSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!emailValue.trim()) return;
    setEmailSubmitted(true);
  };

  const t = (key: string) => {
    const labels: Record<string, Record<string, string>> = {
      title: {
        ja: "ログイン方法を選択",
        en: "Choose a login method",
      },
      google: { ja: "Google でログイン", en: "Sign in with Google" },
      github: { ja: "GitHub でログイン", en: "Sign in with GitHub" },
      discord: { ja: "Discord でログイン", en: "Sign in with Discord" },
      apple: { ja: "Apple でログイン", en: "Sign in with Apple" },
      email: { ja: "Email でログイン", en: "Sign in with Email" },
      close: { ja: "閉じる", en: "Close" },
      back: { ja: "戻る", en: "Back" },
      emailPlaceholder: { ja: "メールアドレス", en: "Email address" },
      sendLink: { ja: "リンクを送信", en: "Send link" },
      emailSent: {
        ja: "送信しました。メールをご確認ください。",
        en: "Sent. Please check your email.",
      },
      emailSentMock: {
        ja: "送信しました。（モックのため実際のメールは送信されません）",
        en: "Sent. (No email is sent in mock mode.)",
      },
      mockLogin: { ja: "モック: ログイン", en: "Mock: Login" },
    };
    return labels[key]?.[locale] ?? labels[key]?.["ja"] ?? key;
  };

  const providerButton = (provider: string) => {
    if (provider === "google") {
      return (
        <button
          key="google"
          type="button"
          className="login-provider-btn login-google-btn"
          onClick={() => handleLogin("google")}
        >
          <GoogleLogoSvg size={20} />
          <span>{t("google")}</span>
        </button>
      );
    }

    return (
      <button
        key={provider}
        type="button"
        className="login-provider-btn"
        onClick={() =>
          provider === "email"
            ? setShowEmailForm(true)
            : handleLogin(provider)
        }
      >
        <span>{t(provider)}</span>
      </button>
    );
  };

  // プロバイダの表示順: google を先頭に固定
  const orderedProviders = [
    "google",
    ...providers.filter((p) => p !== "google"),
  ];

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="login-dialog-title"
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 1000,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "rgba(0,0,0,0.5)",
      }}
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        style={{
          background: "var(--color-bg, #fff)",
          color: "var(--color-fg, #111)",
          padding: "1.5rem",
          borderRadius: "8px",
          minWidth: "280px",
          maxWidth: "90vw",
          width: "360px",
          boxShadow: "0 4px 20px rgba(0,0,0,0.15)",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h2
          id="login-dialog-title"
          style={{ margin: "0 0 1.25rem", fontSize: "1.1rem" }}
        >
          {t("title")}
        </h2>

        {!showEmailForm ? (
          <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
            {orderedProviders.map((p) => providerButton(p))}
            {MOCK_MODE && (
              <button
                type="button"
                className="login-provider-btn"
                onClick={handleMockLogin}
                style={{ borderStyle: "dashed" }}
              >
                {t("mockLogin")}
              </button>
            )}
          </div>
        ) : (
          <div>
            {!emailSubmitted ? (
              <form onSubmit={handleEmailSubmit}>
                <input
                  type="email"
                  value={emailValue}
                  onChange={(e) => setEmailValue(e.target.value)}
                  placeholder={t("emailPlaceholder")}
                  required
                  autoFocus
                  style={{
                    width: "100%",
                    padding: "0.5rem 0.75rem",
                    marginBottom: "1rem",
                    border: "1px solid #ccc",
                    borderRadius: "4px",
                    boxSizing: "border-box",
                  }}
                />
                <div
                  style={{
                    display: "flex",
                    gap: "0.5rem",
                    justifyContent: "flex-end",
                  }}
                >
                  <button
                    type="button"
                    className="btn"
                    onClick={() => setShowEmailForm(false)}
                  >
                    {t("back")}
                  </button>
                  <button type="submit" className="btn btn-primary">
                    {t("sendLink")}
                  </button>
                </div>
              </form>
            ) : (
              <>
                <p style={{ margin: "0 0 1rem" }}>
                  {MOCK_MODE ? t("emailSentMock") : t("emailSent")}
                </p>
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={onClose}
                >
                  {t("close")}
                </button>
              </>
            )}
          </div>
        )}

        <div style={{ marginTop: "1rem", textAlign: "right" }}>
          <button
            type="button"
            className="btn"
            onClick={onClose}
            style={{ fontSize: "0.85rem" }}
          >
            {t("close")}
          </button>
        </div>
      </div>
    </div>
  );
}
