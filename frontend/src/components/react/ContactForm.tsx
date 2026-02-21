import { useState } from 'react';
import { submitContact } from '../../lib/api';

interface Props {
  locale?: string;
  emailLabel: string;
  emailPlaceholder: string;
  nameLabel: string;
  namePlaceholder: string;
  messageLabel: string;
  messagePlaceholder: string;
  submitLabel: string;
  sendingLabel: string;
  successTitle: string;
  successMessage: string;
  errorRequired: string;
  errorFailed: string;
}

export default function ContactForm({
  locale = 'ja',
  emailLabel,
  emailPlaceholder,
  nameLabel,
  namePlaceholder,
  messageLabel,
  messagePlaceholder,
  submitLabel,
  sendingLabel,
  successTitle,
  successMessage,
  errorRequired,
  errorFailed,
}: Props) {
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !message.trim()) {
      setError(errorRequired);
      return;
    }
    setSending(true);
    setError(null);
    try {
      await submitContact({ email: email.trim(), name: name.trim() || undefined, message: message.trim() });
      setSuccess(true);
    } catch {
      setError(errorFailed);
    } finally {
      setSending(false);
    }
  };

  if (success) {
    return (
      <div
        className="card"
        style={{ marginTop: '1.5rem', padding: '1.5rem', textAlign: 'center' }}
      >
        <h2 style={{ marginTop: 0 }}>{successTitle}</h2>
        <p style={{ color: 'var(--color-text-muted, #555)' }}>{successMessage}</p>
      </div>
    );
  }

  return (
    <form className="card" style={{ marginTop: '1.5rem', padding: '1.5rem', display: 'grid', gap: '1rem' }} onSubmit={handleSubmit}>
      {error && (
        <div
          role="alert"
          style={{
            padding: '0.5rem 0.75rem',
            background: 'var(--color-danger, #c00)',
            color: 'white',
            borderRadius: '4px',
            fontSize: '0.9rem',
          }}
        >
          {error}
        </div>
      )}
      <div style={{ display: 'grid', gap: '0.25rem' }}>
        <label htmlFor="contact-email" style={{ fontWeight: 600, fontSize: '0.9rem' }}>
          {emailLabel}
        </label>
        <input
          id="contact-email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={emailPlaceholder}
          required
          style={{
            padding: '0.5rem 0.75rem',
            border: '1px solid var(--color-border, #ccc)',
            borderRadius: '4px',
            fontSize: '1rem',
            width: '100%',
            boxSizing: 'border-box',
          }}
        />
      </div>
      <div style={{ display: 'grid', gap: '0.25rem' }}>
        <label htmlFor="contact-name" style={{ fontWeight: 600, fontSize: '0.9rem' }}>
          {nameLabel}
        </label>
        <input
          id="contact-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={namePlaceholder}
          maxLength={255}
          style={{
            padding: '0.5rem 0.75rem',
            border: '1px solid var(--color-border, #ccc)',
            borderRadius: '4px',
            fontSize: '1rem',
            width: '100%',
            boxSizing: 'border-box',
          }}
        />
      </div>
      <div style={{ display: 'grid', gap: '0.25rem' }}>
        <label htmlFor="contact-message" style={{ fontWeight: 600, fontSize: '0.9rem' }}>
          {messageLabel}
        </label>
        <textarea
          id="contact-message"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder={messagePlaceholder}
          required
          maxLength={5000}
          rows={6}
          style={{
            padding: '0.5rem 0.75rem',
            border: '1px solid var(--color-border, #ccc)',
            borderRadius: '4px',
            fontSize: '1rem',
            width: '100%',
            boxSizing: 'border-box',
            resize: 'vertical',
          }}
        />
        <div style={{ fontSize: '0.8rem', color: 'var(--color-text-muted, #888)', textAlign: 'right' }}>
          {message.length} / 5000
        </div>
      </div>
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button type="submit" className="btn btn-primary" disabled={sending}>
          {sending ? sendingLabel : submitLabel}
        </button>
      </div>
    </form>
  );
}
