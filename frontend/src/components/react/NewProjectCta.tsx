import { useState, useEffect } from "react";
import { getMe } from "../../lib/api";
import LoginDialog from "./LoginDialog";

interface Props {
  locale: string;
  newProjectUrl: string;
}

export default function NewProjectCta({ locale, newProjectUrl }: Props) {
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null);
  const [showLogin, setShowLogin] = useState(false);

  useEffect(() => {
    getMe()
      .then((u) => setLoggedIn(!!u))
      .catch(() => setLoggedIn(false));
  }, []);

  if (loggedIn === null) return null;

  if (loggedIn) {
    return (
      <p style={{ marginTop: "1rem" }}>
        <a href={newProjectUrl} className="btn btn-accent">
          {locale === "ja" ? "新規プロジェクト" : "New Project"}
        </a>
      </p>
    );
  }

  return (
    <>
      <p style={{ marginTop: "1rem", color: "var(--color-text-muted)" }}>
        {locale === "ja"
          ? "ログインして寄付プロジェクトを作成できます。"
          : "Log in to create a donation project."}
        <button
          type="button"
          className="btn btn-primary"
          style={{ marginLeft: "0.75rem" }}
          onClick={() => setShowLogin(true)}
        >
          {locale === "ja" ? "ログイン" : "Login"}
        </button>
      </p>
      <LoginDialog
        open={showLogin}
        locale={locale}
        onClose={() => setShowLogin(false)}
        returnUrl={
          typeof window !== "undefined" ? window.location.pathname : undefined
        }
      />
    </>
  );
}
