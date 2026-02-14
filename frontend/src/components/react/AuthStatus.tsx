import { useState, useEffect } from 'react';
import { getMe, getGoogleLoginUrl, getGitHubLoginUrl, logout, type User } from '../../lib/api';

interface Props {
  locale?: string;
}

export default function AuthStatus({ locale = 'ja' }: Props) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMe()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  const handleGoogleLogin = async () => {
    const { url } = await getGoogleLoginUrl();
    window.location.href = url;
  };

  const handleGitHubLogin = async () => {
    const { url } = await getGitHubLoginUrl();
    window.location.href = url;
  };

  const handleLogout = async () => {
    await logout();
    setUser(null);
  };

  if (loading) {
    return (
      <div className="auth-status">
        <span>{locale === 'ja' ? '読み込み中...' : 'Loading...'}</span>
      </div>
    );
  }

  if (user) {
    return (
      <div className="auth-status">
        <span className="auth-user">
          {user.name || user.email}
        </span>
        <button type="button" className="btn btn-accent" onClick={handleLogout}>
          {locale === 'ja' ? 'ログアウト' : 'Logout'}
        </button>
      </div>
    );
  }

  return (
    <div className="auth-status">
      <button type="button" className="btn btn-primary" onClick={handleGoogleLogin}>
        {locale === 'ja' ? 'Google' : 'Google'}
      </button>
      <button type="button" className="btn btn-primary" onClick={handleGitHubLogin}>
        {locale === 'ja' ? 'GitHub' : 'GitHub'}
      </button>
    </div>
  );
}
