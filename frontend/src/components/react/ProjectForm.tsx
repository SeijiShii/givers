import { useState } from 'react';
import type { Project, ProjectCosts, ProjectAlerts, AmountInputType } from '../../lib/api';
import { createProject, updateProject } from '../../lib/api';
import { t, type Locale } from '../../lib/i18n';

interface Props {
  locale: Locale;
  project?: Project | null;
  redirectPath: string;
}

const defaultCosts: ProjectCosts = {
  server_cost_monthly: 0,
  dev_cost_per_day: 0,
  dev_days_per_month: 0,
  other_cost_monthly: 0,
};

const defaultAlerts: ProjectAlerts = {
  warning_threshold: 60,
  critical_threshold: 30,
};

function monthlyTargetFromCosts(c: ProjectCosts): number {
  return c.server_cost_monthly + c.dev_cost_per_day * c.dev_days_per_month + c.other_cost_monthly;
}

export default function ProjectForm({ locale, project, redirectPath }: Props) {
  const isEdit = !!project;

  const [amountType, setAmountType] = useState<AmountInputType>(() => {
    if (!project) return 'want';
    const hasWant = project.owner_want_monthly != null && project.owner_want_monthly > 0;
    const hasCost = project.costs && monthlyTargetFromCosts(project.costs) > 0;
    if (hasWant && hasCost) return 'both';
    if (hasCost) return 'cost';
    return 'want';
  });

  const [name, setName] = useState(project?.name ?? '');
  const [description, setDescription] = useState(project?.description ?? '');
  const [ownerWant, setOwnerWant] = useState(project?.owner_want_monthly ?? 0);
  const [costs, setCosts] = useState<ProjectCosts>(project?.costs ?? defaultCosts);

  // 期限
  const [deadlineType, setDeadlineType] = useState<'permanent' | 'date'>(
    project?.deadline ? 'date' : 'permanent'
  );
  const [deadlineValue, setDeadlineValue] = useState(
    project?.deadline ? project.deadline.slice(0, 10) : ''
  );

  // アラート閾値
  const [alertsEnabled, setAlertsEnabled] = useState(!!project?.alerts);
  const [alerts, setAlerts] = useState<ProjectAlerts>(project?.alerts ?? defaultAlerts);

  // Stripe Connect（新規作成のみ）
  const [stripeConnecting, setStripeConnecting] = useState(false);
  const [stripeConnected, setStripeConnected] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleStripeConnect = () => {
    setStripeConnecting(true);
    // モック: 1.5秒後に接続完了
    setTimeout(() => {
      setStripeConnecting(false);
      setStripeConnected(true);
    }, 1500);
  };

  const canSubmit = isEdit || stripeConnected;

  const doSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setError(null);
    setSubmitting(true);
    try {
      const payload = {
        name,
        description,
        deadline: deadlineType === 'date' && deadlineValue ? deadlineValue : null,
        owner_want_monthly:
          amountType === 'want' || amountType === 'both'
            ? ownerWant > 0
              ? ownerWant
              : null
            : null,
        costs: amountType === 'cost' || amountType === 'both' ? costs : null,
        alerts: alertsEnabled ? alerts : null,
      };
      if (isEdit) {
        await updateProject(project!.id, payload);
      } else {
        await createProject(payload);
      }
      window.location.href = redirectPath;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={doSubmit} className="card" style={{ maxWidth: '32rem', marginTop: '1.5rem' }}>
      {/* プロジェクト名 */}
      <div style={{ marginBottom: '1rem' }}>
        <label htmlFor="name" style={{ display: 'block', marginBottom: '0.25rem' }}>
          プロジェクト名 *
        </label>
        <input
          id="name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          style={{ width: '100%', padding: '0.5rem' }}
        />
      </div>

      {/* 説明 */}
      <div style={{ marginBottom: '1rem' }}>
        <label htmlFor="description" style={{ display: 'block', marginBottom: '0.25rem' }}>
          説明
        </label>
        <textarea
          id="description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={4}
          style={{ width: '100%', padding: '0.5rem' }}
        />
      </div>

      {/* 期限 */}
      <fieldset style={{ marginBottom: '1rem', border: '1px solid var(--color-border)', padding: '1rem', borderRadius: '4px' }}>
        <legend>{t(locale, 'projects.deadline')}</legend>
        <label style={{ display: 'block', marginBottom: '0.5rem' }}>
          <input
            type="radio"
            name="deadlineType"
            checked={deadlineType === 'permanent'}
            onChange={() => setDeadlineType('permanent')}
          />
          {' '}{t(locale, 'projects.deadlinePermanent')}
        </label>
        <label style={{ display: 'block' }}>
          <input
            type="radio"
            name="deadlineType"
            checked={deadlineType === 'date'}
            onChange={() => setDeadlineType('date')}
          />
          {' '}{t(locale, 'projects.deadlineDate')}
        </label>
        {deadlineType === 'date' && (
          <input
            type="date"
            value={deadlineValue}
            onChange={(e) => setDeadlineValue(e.target.value)}
            style={{ marginTop: '0.5rem', padding: '0.5rem' }}
          />
        )}
      </fieldset>

      {/* 金額タイプ */}
      <fieldset style={{ marginBottom: '1rem', border: '1px solid var(--color-border)', padding: '1rem', borderRadius: '4px' }}>
        <legend>{t(locale, 'projects.amountType')}</legend>
        <label style={{ display: 'block', marginBottom: '0.5rem' }}>
          <input
            type="radio"
            name="amountType"
            checked={amountType === 'want'}
            onChange={() => setAmountType('want')}
          />
          {' '}{t(locale, 'projects.amountTypeWant')}
        </label>
        <label style={{ display: 'block', marginBottom: '0.5rem' }}>
          <input
            type="radio"
            name="amountType"
            checked={amountType === 'cost'}
            onChange={() => setAmountType('cost')}
          />
          {' '}{t(locale, 'projects.amountTypeCost')}
        </label>
        <label style={{ display: 'block' }}>
          <input
            type="radio"
            name="amountType"
            checked={amountType === 'both'}
            onChange={() => setAmountType('both')}
          />
          {' '}{t(locale, 'projects.amountTypeBoth')}
        </label>
      </fieldset>

      {/* 最低希望額 */}
      {(amountType === 'want' || amountType === 'both') && (
        <div style={{ marginBottom: '1rem' }}>
          <label htmlFor="ownerWant" style={{ display: 'block', marginBottom: '0.25rem' }}>
            {t(locale, 'projects.ownerWant')} (¥)
          </label>
          <input
            id="ownerWant"
            type="number"
            min={0}
            value={ownerWant || ''}
            onChange={(e) => setOwnerWant(parseInt(e.target.value, 10) || 0)}
            style={{ width: '100%', padding: '0.5rem' }}
          />
        </div>
      )}

      {/* コスト内訳 */}
      {(amountType === 'cost' || amountType === 'both') && (
        <div style={{ marginBottom: '1rem', padding: '1rem', border: '1px solid var(--color-border)', borderRadius: '4px' }}>
          <h3 style={{ marginTop: 0 }}>{t(locale, 'projects.costBreakdown')}</h3>
          <div style={{ marginBottom: '0.5rem' }}>
            <label htmlFor="serverCost" style={{ display: 'block', marginBottom: '0.25rem' }}>
              {t(locale, 'projects.serverCost')} (¥)
            </label>
            <input
              id="serverCost"
              type="number"
              min={0}
              value={costs.server_cost_monthly || ''}
              onChange={(e) => setCosts({ ...costs, server_cost_monthly: parseInt(e.target.value, 10) || 0 })}
              style={{ width: '100%', padding: '0.5rem' }}
            />
          </div>
          <div style={{ marginBottom: '0.5rem' }}>
            <label htmlFor="devCostPerDay" style={{ display: 'block', marginBottom: '0.25rem' }}>
              {t(locale, 'projects.devCostPerDay')} (¥)
            </label>
            <input
              id="devCostPerDay"
              type="number"
              min={0}
              value={costs.dev_cost_per_day || ''}
              onChange={(e) => setCosts({ ...costs, dev_cost_per_day: parseInt(e.target.value, 10) || 0 })}
              style={{ width: '100%', padding: '0.5rem' }}
            />
          </div>
          <div style={{ marginBottom: '0.5rem' }}>
            <label htmlFor="devDaysPerMonth" style={{ display: 'block', marginBottom: '0.25rem' }}>
              {t(locale, 'projects.devDaysPerMonth')}
            </label>
            <input
              id="devDaysPerMonth"
              type="number"
              min={0}
              value={costs.dev_days_per_month || ''}
              onChange={(e) => setCosts({ ...costs, dev_days_per_month: parseInt(e.target.value, 10) || 0 })}
              style={{ width: '100%', padding: '0.5rem' }}
            />
          </div>
          <div>
            <label htmlFor="otherCost" style={{ display: 'block', marginBottom: '0.25rem' }}>
              {t(locale, 'projects.otherCost')} (¥)
            </label>
            <input
              id="otherCost"
              type="number"
              min={0}
              value={costs.other_cost_monthly || ''}
              onChange={(e) => setCosts({ ...costs, other_cost_monthly: parseInt(e.target.value, 10) || 0 })}
              style={{ width: '100%', padding: '0.5rem' }}
            />
          </div>
        </div>
      )}

      {/* アラート閾値 */}
      <fieldset style={{ marginBottom: '1rem', border: '1px solid var(--color-border)', padding: '1rem', borderRadius: '4px' }}>
        <legend>{t(locale, 'projects.alertsTitle')}</legend>
        <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.75rem', cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={alertsEnabled}
            onChange={(e) => setAlertsEnabled(e.target.checked)}
          />
          アラートを設定する
        </label>
        {alertsEnabled && (
          <>
            <p style={{ margin: '0 0 0.75rem', fontSize: '0.85rem', color: 'var(--color-text-muted)' }}>
              {t(locale, 'projects.alertsHint')}
            </p>
            <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
              <div style={{ flex: 1, minWidth: '120px' }}>
                <label htmlFor="warningThreshold" style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9rem' }}>
                  {t(locale, 'projects.warningThreshold')}
                </label>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                  <input
                    id="warningThreshold"
                    type="number"
                    min={1}
                    max={100}
                    value={alerts.warning_threshold || ''}
                    onChange={(e) => setAlerts({ ...alerts, warning_threshold: parseInt(e.target.value, 10) || 0 })}
                    style={{ width: '80px', padding: '0.5rem' }}
                  />
                  <span>%</span>
                </div>
              </div>
              <div style={{ flex: 1, minWidth: '120px' }}>
                <label htmlFor="criticalThreshold" style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9rem' }}>
                  {t(locale, 'projects.criticalThreshold')}
                </label>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                  <input
                    id="criticalThreshold"
                    type="number"
                    min={1}
                    max={100}
                    value={alerts.critical_threshold || ''}
                    onChange={(e) => setAlerts({ ...alerts, critical_threshold: parseInt(e.target.value, 10) || 0 })}
                    style={{ width: '80px', padding: '0.5rem' }}
                  />
                  <span>%</span>
                </div>
              </div>
            </div>
          </>
        )}
      </fieldset>

      {/* Stripe Connect（新規作成のみ） */}
      {!isEdit && (
        <div style={{
          marginBottom: '1.5rem',
          padding: '1rem',
          border: `1px solid ${stripeConnected ? 'var(--color-primary)' : 'var(--color-border)'}`,
          borderRadius: '4px',
          background: stripeConnected ? 'var(--color-primary-muted, rgba(99,102,241,0.06))' : undefined,
        }}>
          <h3 style={{ marginTop: 0 }}>{t(locale, 'projects.stripeConnectTitle')}</h3>
          {!stripeConnected ? (
            <>
              <p style={{ marginBottom: '0.75rem', fontSize: '0.9rem' }}>
                {t(locale, 'projects.stripeConnectDesc')}
              </p>
              <button
                type="button"
                className="btn btn-primary"
                disabled={stripeConnecting}
                onClick={handleStripeConnect}
              >
                {stripeConnecting ? t(locale, 'projects.stripeConnecting') : t(locale, 'projects.stripeConnectButton')}
              </button>
              <p style={{ marginTop: '0.5rem', marginBottom: 0, fontSize: '0.8rem', color: 'var(--color-text-muted)' }}>
                {t(locale, 'projects.stripeConnectNote')}
              </p>
            </>
          ) : (
            <p style={{ margin: 0, fontWeight: 600, color: 'var(--color-primary)' }}>
              {t(locale, 'projects.stripeConnected')}
            </p>
          )}
        </div>
      )}

      {error && <p style={{ color: 'var(--color-danger)', marginBottom: '1rem' }}>{error}</p>}

      <button
        type="submit"
        className="btn btn-accent"
        disabled={submitting || !canSubmit}
        title={!canSubmit ? 'Stripe アカウントを接続してから公開できます' : undefined}
      >
        {submitting
          ? t(locale, 'projects.loading')
          : isEdit
            ? t(locale, 'projects.editProject')
            : t(locale, 'projects.newProject')}
      </button>

      {!isEdit && !stripeConnected && (
        <p style={{ marginTop: '0.5rem', marginBottom: 0, fontSize: '0.85rem', color: 'var(--color-text-muted)' }}>
          Stripe アカウントを接続するとプロジェクトを公開できます。
        </p>
      )}
    </form>
  );
}
