import { useState, useEffect } from 'react';
import { getMe } from '../../lib/api';
import { t, type Locale } from '../../lib/i18n';

interface Props {
  locale: Locale;
  basePath: string;
}

export default function NavAdminLink({ locale, basePath }: Props) {
  const [isHost, setIsHost] = useState<boolean | null>(null);

  useEffect(() => {
    const fetchRole = () => {
      getMe()
        .then((user) => setIsHost(user?.role === 'host'))
        .catch(() => setIsHost(false));
    };
    fetchRole();
    if (typeof window === 'undefined') return;
    window.addEventListener('givers-mock-login-changed', fetchRole);
    return () => window.removeEventListener('givers-mock-login-changed', fetchRole);
  }, []);

  if (isHost !== true) return null;

  return (
    <a href={`${basePath}/admin/users`}>{t(locale, 'nav.adminUsers')}</a>
  );
}
