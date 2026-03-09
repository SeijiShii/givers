import { useEffect, useState } from 'react';
import { getMe, getAdminAlerts, updateAlertStatus, type User, type DonationAlert } from '../../lib/api';
import type { Locale } from '../../lib/i18n';

const PAGE_SIZE = 50;

const ALERT_TYPE_KEYS: Record<string, string> = {
  self_dealing: 'admin.alertTypeSelfDealing',
  high_frequency: 'admin.alertTypeHighFrequency',
  high_amount: 'admin.alertTypeHighAmount',
  round_tripping: 'admin.alertTypeRoundTripping',
};

interface Props {
  locale: Locale;
  title: string;
  forbiddenMessage: string;
  emptyLabel: string;
  filterAllLabel: string;
  filterNewLabel: string;
  filterAcknowledgedLabel: string;
  filterResolvedLabel: string;
  statusNewLabel: string;
  statusAcknowledgedLabel: string;
  statusResolvedLabel: string;
  acknowledgeLabel: string;
  resolveLabel: string;
  reopenLabel: string;
  typeSelfDealingLabel: string;
  typeHighFrequencyLabel: string;
  typeHighAmountLabel: string;
  typeRoundTrippingLabel: string;
  severityWarningLabel: string;
  severityCriticalLabel: string;
  projectLabel: string;
  donorLabel: string;
  loadingLabel: string;
  loadMoreLabel: string;
}

const typeLabels: Record<string, string> = {};
const severityColors: Record<string, { bg: string; color: string }> = {
  warning: { bg: '#fff3cd', color: '#856404' },
  critical: { bg: '#f8d7da', color: '#721c24' },
};
const statusColors: Record<string, { bg: string; color: string }> = {
  new: { bg: 'var(--color-primary, #007bff)', color: 'white' },
  acknowledged: { bg: '#ffc107', color: '#333' },
  resolved: { bg: 'var(--color-muted, #eee)', color: 'var(--color-text-muted, #555)' },
};

export default function AdminAlertList({
  locale,
  title,
  forbiddenMessage,
  emptyLabel,
  filterAllLabel,
  filterNewLabel,
  filterAcknowledgedLabel,
  filterResolvedLabel,
  statusNewLabel,
  statusAcknowledgedLabel,
  statusResolvedLabel,
  acknowledgeLabel,
  resolveLabel,
  reopenLabel,
  typeSelfDealingLabel,
  typeHighFrequencyLabel,
  typeHighAmountLabel,
  typeRoundTrippingLabel,
  severityWarningLabel,
  severityCriticalLabel,
  projectLabel,
  donorLabel,
  loadingLabel,
  loadMoreLabel,
}: Props) {
  const [me, setMe] = useState<User | null>(null);
  const [alerts, setAlerts] = useState<DonationAlert[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('new');
  const [offset, setOffset] = useState(0);
  const [updatingIds, setUpdatingIds] = useState<Set<string>>(new Set());

  const alertTypeLabels: Record<string, string> = {
    self_dealing: typeSelfDealingLabel,
    high_frequency: typeHighFrequencyLabel,
    high_amount: typeHighAmountLabel,
    round_tripping: typeRoundTrippingLabel,
  };
  const severityLabels: Record<string, string> = {
    warning: severityWarningLabel,
    critical: severityCriticalLabel,
  };
  const statusLabels: Record<string, string> = {
    new: statusNewLabel,
    acknowledged: statusAcknowledgedLabel,
    resolved: statusResolvedLabel,
  };

  useEffect(() => {
    getMe()
      .then(setMe)
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, []);

  const fetchAlerts = (status: string, off: number, append: boolean) => {
    getAdminAlerts({
      status: status === 'all' ? undefined : status,
      limit: PAGE_SIZE,
      offset: off,
    })
      .then((data) => {
        setAlerts((prev) => (append ? [...prev, ...data.alerts] : data.alerts));
        setTotal(data.total);
      })
      .catch(() => {
        if (!append) {
          setAlerts([]);
          setTotal(0);
        }
      });
  };

  useEffect(() => {
    if (me?.role !== 'host') return;
    setOffset(0);
    fetchAlerts(filter, 0, false);
  }, [me?.role, filter]);

  const handleStatusChange = async (alert: DonationAlert, newStatus: string) => {
    setUpdatingIds((prev) => new Set(prev).add(alert.id));
    try {
      await updateAlertStatus(alert.id, newStatus);
      setAlerts((prev) =>
        prev.map((a) => (a.id === alert.id ? { ...a, status: newStatus } : a)),
      );
    } finally {
      setUpdatingIds((prev) => {
        const next = new Set(prev);
        next.delete(alert.id);
        return next;
      });
    }
  };

  const handleLoadMore = () => {
    const newOffset = offset + PAGE_SIZE;
    setOffset(newOffset);
    fetchAlerts(filter, newOffset, true);
  };

  if (loading) {
    return <p>{loadingLabel}</p>;
  }

  if (me?.role !== 'host') {
    return (
      <div className="card" style={{ marginTop: '1rem', padding: '1rem' }}>
        <p style={{ color: 'var(--color-danger, #c00)' }}>{forbiddenMessage}</p>
      </div>
    );
  }

  const parseDetails = (details: string): Record<string, unknown> => {
    try {
      return JSON.parse(details);
    } catch {
      return {};
    }
  };

  const formatDetails = (alert: DonationAlert): string => {
    const d = parseDetails(alert.details);
    const parts: string[] = [];
    if (d.amount != null) parts.push(`${Number(d.amount).toLocaleString()} JPY`);
    if (d.daily_total != null) parts.push(`${locale === 'ja' ? '日計' : 'Daily'}: ${Number(d.daily_total).toLocaleString()} JPY`);
    if (d.count != null) parts.push(`${d.count}${locale === 'ja' ? '回' : 'x'}`);
    if (d.window_minutes != null) parts.push(`${d.window_minutes}min`);
    if (d.refund_count != null) parts.push(`${locale === 'ja' ? '返金' : 'Refunds'}: ${d.refund_count}`);
    if (d.threshold != null) parts.push(`${locale === 'ja' ? '閾値' : 'Threshold'}: ${Number(d.threshold).toLocaleString()}`);
    return parts.join(' / ');
  };

  const filters = [
    { key: 'all', label: filterAllLabel },
    { key: 'new', label: filterNewLabel },
    { key: 'acknowledged', label: filterAcknowledgedLabel },
    { key: 'resolved', label: filterResolvedLabel },
  ];

  const newCount = alerts.filter((a) => a.status === 'new').length;

  return (
    <div style={{ marginTop: '1rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
        <h1 style={{ margin: 0 }}>
          {title}
          {filter === 'new' && total > 0 && (
            <span
              style={{
                marginLeft: '0.5rem',
                fontSize: '0.8rem',
                background: 'var(--color-danger, #c00)',
                color: 'white',
                borderRadius: '9999px',
                padding: '0.1rem 0.5rem',
                verticalAlign: 'middle',
              }}
            >
              {total}
            </span>
          )}
        </h1>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          {filters.map((f) => (
            <button
              key={f.key}
              type="button"
              className={`btn${filter === f.key ? ' btn-primary' : ''}`}
              onClick={() => setFilter(f.key)}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {alerts.length === 0 ? (
        <div className="card" style={{ padding: '1.5rem', color: 'var(--color-text-muted, #888)' }}>
          {emptyLabel}
        </div>
      ) : (
        <div style={{ display: 'grid', gap: '0.75rem' }}>
          {alerts.map((alert) => {
            const sevStyle = severityColors[alert.severity] ?? severityColors.warning;
            const statStyle = statusColors[alert.status] ?? statusColors.new;

            return (
              <div
                key={alert.id}
                className="card"
                style={{
                  padding: '1rem 1.25rem',
                  borderLeft: `4px solid ${alert.severity === 'critical' ? '#dc3545' : '#ffc107'}`,
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '0.5rem', flexWrap: 'wrap' }}>
                  <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap' }}>
                    <span
                      style={{
                        fontSize: '0.75rem',
                        padding: '0.1rem 0.5rem',
                        borderRadius: '4px',
                        background: sevStyle.bg,
                        color: sevStyle.color,
                        fontWeight: 'bold',
                      }}
                    >
                      {severityLabels[alert.severity] ?? alert.severity}
                    </span>
                    <span style={{ fontWeight: 'bold' }}>
                      {alertTypeLabels[alert.alert_type] ?? alert.alert_type}
                    </span>
                    <span
                      style={{
                        fontSize: '0.75rem',
                        padding: '0.1rem 0.5rem',
                        borderRadius: '4px',
                        background: statStyle.bg,
                        color: statStyle.color,
                      }}
                    >
                      {statusLabels[alert.status] ?? alert.status}
                    </span>
                  </div>
                  <div style={{ display: 'flex', gap: '0.25rem' }}>
                    {alert.status === 'new' && (
                      <>
                        <button
                          type="button"
                          className="btn"
                          style={{ fontSize: '0.8rem', padding: '0.15rem 0.5rem' }}
                          disabled={updatingIds.has(alert.id)}
                          onClick={() => handleStatusChange(alert, 'acknowledged')}
                        >
                          {acknowledgeLabel}
                        </button>
                        <button
                          type="button"
                          className="btn"
                          style={{ fontSize: '0.8rem', padding: '0.15rem 0.5rem' }}
                          disabled={updatingIds.has(alert.id)}
                          onClick={() => handleStatusChange(alert, 'resolved')}
                        >
                          {resolveLabel}
                        </button>
                      </>
                    )}
                    {alert.status === 'acknowledged' && (
                      <button
                        type="button"
                        className="btn"
                        style={{ fontSize: '0.8rem', padding: '0.15rem 0.5rem' }}
                        disabled={updatingIds.has(alert.id)}
                        onClick={() => handleStatusChange(alert, 'resolved')}
                      >
                        {resolveLabel}
                      </button>
                    )}
                    {alert.status === 'resolved' && (
                      <button
                        type="button"
                        className="btn"
                        style={{ fontSize: '0.8rem', padding: '0.15rem 0.5rem' }}
                        disabled={updatingIds.has(alert.id)}
                        onClick={() => handleStatusChange(alert, 'new')}
                      >
                        {reopenLabel}
                      </button>
                    )}
                  </div>
                </div>

                <div style={{ marginTop: '0.5rem', fontSize: '0.9rem', color: 'var(--color-text-muted, #555)', display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
                  <span><strong>{projectLabel}:</strong> {alert.project_id}</span>
                  <span><strong>{donorLabel}:</strong> {alert.donor_type === 'user' ? alert.donor_id : `${locale === 'ja' ? '匿名' : 'Anonymous'} (${alert.donor_id.slice(0, 8)}...)`}</span>
                  <span>{new Date(alert.created_at).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')}</span>
                </div>

                {formatDetails(alert) && (
                  <div style={{ marginTop: '0.4rem', fontSize: '0.85rem', color: 'var(--color-text-muted, #777)' }}>
                    {formatDetails(alert)}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {alerts.length < total && (
        <div style={{ textAlign: 'center', marginTop: '1rem' }}>
          <button type="button" className="btn" onClick={handleLoadMore}>
            {loadMoreLabel}
          </button>
        </div>
      )}
    </div>
  );
}
