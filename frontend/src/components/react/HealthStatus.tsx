import { useState, useEffect } from 'react';
import { healthCheck } from '../../lib/api';

interface Props {
  label?: string;
}

export default function HealthStatus({ label = 'API ステータス' }: Props) {
  const [status, setStatus] = useState<string>('loading');
  const [message, setMessage] = useState<string>('');

  useEffect(() => {
    healthCheck()
      .then((res) => {
        setStatus(res.status);
        setMessage(res.message);
      })
      .catch((err) => {
        setStatus('error');
        setMessage(err.message);
      });
  }, []);

  return (
    <div className="status-box" data-status={status}>
      <strong>{label}:</strong> {status}
      {message && <span> — {message}</span>}
    </div>
  );
}
