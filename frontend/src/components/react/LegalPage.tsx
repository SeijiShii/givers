import { useState, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import { getLegalDoc } from '../../lib/api';

interface Props {
  type: 'terms' | 'privacy' | 'disclaimer';
  notConfiguredLabel: string;
  backToTopLabel: string;
  backToTopHref: string;
}

export default function LegalPage({ type, notConfiguredLabel, backToTopLabel, backToTopHref }: Props) {
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getLegalDoc(type)
      .then((doc) => setContent(doc?.content ?? null))
      .catch(() => setContent(null))
      .finally(() => setLoading(false));
  }, [type]);

  if (loading) {
    return <div className="card" style={{ marginTop: '1.5rem', padding: '1.5rem' }}>...</div>;
  }

  if (!content) {
    return (
      <div className="card" style={{ marginTop: '1.5rem', padding: '1.5rem' }}>
        <p style={{ color: 'var(--color-text-muted, #888)' }}>{notConfiguredLabel}</p>
        <a href={backToTopHref} style={{ fontSize: '0.9rem' }}>{backToTopLabel}</a>
      </div>
    );
  }

  return (
    <div
      className="card"
      style={{
        marginTop: '1.5rem',
        padding: '1.5rem',
        lineHeight: 1.7,
        maxWidth: '720px',
      }}
    >
      <ReactMarkdown>{content}</ReactMarkdown>
    </div>
  );
}
