import { useState } from 'react';
import type { Project, ProjectCosts, AmountInputType } from '../../lib/api';
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
  const [costs, setCosts] = useState<ProjectCosts>(
    project?.costs ?? defaultCosts
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const doSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const payload = {
        name,
        description,
        owner_want_monthly: amountType === 'want' || amountType === 'both' ? (ownerWant > 0 ? ownerWant : null) : null,
        costs: amountType === 'cost' || amountType === 'both' ? costs : null,
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

      {error && <p style={{ color: 'var(--color-danger)', marginBottom: '1rem' }}>{error}</p>}
      <button type="submit" className="btn btn-accent" disabled={submitting}>
        {submitting ? t(locale, 'projects.loading') : isEdit ? t(locale, 'projects.editProject') : t(locale, 'projects.newProject')}
      </button>
    </form>
  );
}
