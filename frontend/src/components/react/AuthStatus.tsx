import { useState, useEffect } from 'react';
import {
  getMe,
  getGoogleLoginUrl,
  getGitHubLoginUrl,
  getAppleLoginUrl,
  logout,
  MOCK_LOGIN_MODE_KEY,
  type User,
} from '../../lib/api';

const MOCK_MODE = import.meta.env.PUBLIC_MOCK_MODE === 'true';

interface Props {
  locale?: string;
}

export default function AuthStatus({ locale = 'ja' }: Props) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [showEmailModal, setShowEmailModal] = useState(false);
  const [emailValue, setEmailValue] = useState('');
  const [emailSubmitted, setEmailSubmitted] = useState(false);

  const fetchUser = () => {
    getMe()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchUser();
  }, []);

  const handleMockModeSwitch = (mode: 'host' | 'donor' | 'project_owner') => {
    if (typeof window === 'undefined' || !window.localStorage) return;
    window.localStorage.setItem(MOCK_LOGIN_MODE_KEY, mode);
    setLoading(true);
    getMe()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
    window.dispatchEvent(new CustomEvent('givers-mock-login-changed'));
  };

  const handleLogout = async () => {
    if (MOCK_MODE && typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem(MOCK_LOGIN_MODE_KEY, 'logout');
      setUser(null);
      return;
    }
    await logout();
    setUser(null);
  };

  const handleGoogleLogin = async () => {
    const { url } = await getGoogleLoginUrl();
    window.location.href = url;
  };

  const handleGitHubLogin = async () => {
    const { url } = await getGitHubLoginUrl();
    window.location.href = url;
  };

  const handleAppleLogin = async () => {
    const { url } = await getAppleLoginUrl();
    window.location.href = url;
  };

  const handleEmailLogin = () => {
    setEmailValue('');
    setEmailSubmitted(false);
    setShowEmailModal(true);
  };

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!emailValue.trim()) return;
    // モックでは送信 API なし。本番では POST /api/auth/magic-link 等を呼ぶ想定。
    setEmailSubmitted(true);
  };

  const closeEmailModal = () => {
    setShowEmailModal(false);
    setEmailValue('');
    setEmailSubmitted(false);
  };

  if (loading) {
    return (
      <div className="auth-status">
        <span>{locale === 'ja' ? '読み込み中...' : 'Loading...'}</span>
      </div>
    );
  }

  if (user) {
    const mockMode = typeof window !== 'undefined' && window.localStorage
      ? window.localStorage.getItem(MOCK_LOGIN_MODE_KEY) ?? 'host'
      : 'host';
    const roleLabel = user.role === 'host'
      ? (locale === 'ja' ? 'ホスト' : 'Host')
      : user.role === 'donor'
        ? (locale === 'ja' ? '寄付メンバー' : 'Donor')
        : (locale === 'ja' ? 'プロジェクトオーナー' : 'Project Owner');
    return (
      <div className="auth-status">
        {MOCK_MODE && (
          <div className="mock-login-switch" style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
            <button
              type="button"
              onClick={() => handleMockModeSwitch('host')}
              style={{
                padding: '0.25rem 0.5rem',
                fontSize: '0.75rem',
                border: '1px solid rgba(255,255,255,0.5)',
                borderRadius: '4px',
                background: mockMode === 'host' ? 'rgba(255,255,255,0.25)' : 'transparent',
                color: 'inherit',
                cursor: 'pointer',
              }}
            >
              {locale === 'ja' ? 'ホスト' : 'Host'}
            </button>
            <button
              type="button"
              onClick={() => handleMockModeSwitch('donor')}
              style={{
                padding: '0.25rem 0.5rem',
                fontSize: '0.75rem',
                border: '1px solid rgba(255,255,255,0.5)',
                borderRadius: '4px',
                background: mockMode === 'donor' ? 'rgba(255,255,255,0.25)' : 'transparent',
                color: 'inherit',
                cursor: 'pointer',
              }}
            >
              {locale === 'ja' ? '寄付メンバー' : 'Donor'}
            </button>
            <button
              type="button"
              onClick={() => handleMockModeSwitch('project_owner')}
              style={{
                padding: '0.25rem 0.5rem',
                fontSize: '0.75rem',
                border: '1px solid rgba(255,255,255,0.5)',
                borderRadius: '4px',
                background: mockMode === 'project_owner' || mockMode === 'member' ? 'rgba(255,255,255,0.25)' : 'transparent',
                color: 'inherit',
                cursor: 'pointer',
              }}
            >
              {locale === 'ja' ? 'プロジェクトオーナー' : 'Project Owner'}
            </button>
          </div>
        )}
        <span className="auth-user">
          {user.name || user.email}
          {user.role && (
            <span style={{ marginLeft: '0.25rem', fontSize: '0.75rem', opacity: 0.9 }}>
              ({roleLabel})
            </span>
          )}
        </span>
        <button type="button" className="btn btn-accent" onClick={handleLogout}>
          {locale === 'ja' ? 'ログアウト' : 'Logout'}
        </button>
      </div>
    );
  }

  return (
    <>
      <div className="auth-status">
        {MOCK_MODE && (
          <button
            type="button"
            onClick={() => {
              if (typeof window !== 'undefined' && window.localStorage) {
                window.localStorage.setItem(MOCK_LOGIN_MODE_KEY, 'host');
                setLoading(true);
                getMe()
                  .then(setUser)
                  .catch(() => setUser(null))
                  .finally(() => setLoading(false));
                window.dispatchEvent(new CustomEvent('givers-mock-login-changed'));
              }
            }}
            className="btn btn-primary"
            style={{ fontSize: '0.85rem', padding: '0.3rem 0.6rem' }}
          >
            {locale === 'ja' ? 'モック: ログイン' : 'Mock: Login'}
          </button>
        )}
        {!MOCK_MODE && (
          <>
            <button type="button" className="btn btn-primary" onClick={handleGoogleLogin}>
              Google
            </button>
            <button type="button" className="btn btn-primary" onClick={handleGitHubLogin}>
              GitHub
            </button>
            <button type="button" className="btn btn-primary" onClick={handleAppleLogin}>
              Apple
            </button>
            <button type="button" className="btn btn-primary" onClick={handleEmailLogin}>
              Email
            </button>
          </>
        )}
        {MOCK_MODE && (
          <button type="button" className="btn btn-primary" onClick={handleEmailLogin}>
            Email
          </button>
        )}
      </div>
      {showEmailModal && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="email-login-title"
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 1000,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'rgba(0,0,0,0.5)',
          }}
          onClick={(e) => e.target === e.currentTarget && closeEmailModal()}
        >
          <div
            style={{
              background: 'var(--color-bg, #fff)',
              color: 'var(--color-fg, #111)',
              padding: '1.5rem',
              borderRadius: '8px',
              minWidth: '280px',
              maxWidth: '90vw',
              boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="email-login-title" style={{ margin: '0 0 1rem', fontSize: '1.1rem' }}>
              {locale === 'ja' ? 'Email でログイン' : 'Log in with Email'}
            </h2>
            {!emailSubmitted ? (
              <form onSubmit={handleEmailSubmit}>
                <input
                  type="email"
                  value={emailValue}
                  onChange={(e) => setEmailValue(e.target.value)}
                  placeholder={locale === 'ja' ? 'メールアドレス' : 'Email address'}
                  required
                  autoFocus
                  style={{
                    width: '100%',
                    padding: '0.5rem 0.75rem',
                    marginBottom: '1rem',
                    border: '1px solid #ccc',
                    borderRadius: '4px',
                    boxSizing: 'border-box',
                  }}
                />
                <div style={{ display: 'flex', gap: '0.5rem', justifyContent: 'flex-end' }}>
                  <button type="button" className="btn" onClick={closeEmailModal}>
                    {locale === 'ja' ? '閉じる' : 'Close'}
                  </button>
                  <button type="submit" className="btn btn-primary">
                    {locale === 'ja' ? 'リンクを送信' : 'Send link'}
                  </button>
                </div>
              </form>
            ) : (
              <>
                <p style={{ margin: '0 0 1rem' }}>
                  {MOCK_MODE
                    ? (locale === 'ja' ? '送信しました。（モックのため実際のメールは送信されません）' : 'Sent. (No email is sent in mock mode.)')
                    : (locale === 'ja' ? '送信しました。メールをご確認ください。' : 'Sent. Please check your email.')}
                </p>
                <button type="button" className="btn btn-primary" onClick={closeEmailModal}>
                  {locale === 'ja' ? '閉じる' : 'Close'}
                </button>
              </>
            )}
          </div>
        </div>
      )}
    </>
  );
}
