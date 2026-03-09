import { useState, useEffect } from 'react';
import { getMe } from '../../lib/api';
import { t, type Locale } from '../../lib/i18n';

interface Props {
  locale: Locale;
  basePath: string;
}

export default function NavMessageLink({ locale, basePath }: Props) {
  const [role, setRole] = useState<'host' | 'user' | null>(null);

  useEffect(() => {
    const fetchRole = () => {
      getMe()
        .then((user) => {
          if (!user) setRole(null);
          else setRole(user.role === 'host' ? 'host' : 'user');
        })
        .catch(() => setRole(null));
    };
    fetchRole();
    if (typeof window === 'undefined') return;
    window.addEventListener('givers-mock-login-changed', fetchRole);
    return () => window.removeEventListener('givers-mock-login-changed', fetchRole);
  }, []);

  if (role === null) return null;

  if (role === 'host') {
    return (
      <a href={`${basePath}/admin/messages`}>{t(locale, 'nav.adminMessages')}</a>
    );
  }

  return (
    <a href={`${basePath}/messages`}>{t(locale, 'nav.messages')}</a>
  );
}
