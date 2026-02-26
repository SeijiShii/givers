import { useState } from "react";
import type {
  Project,
  CostItem,
  ProjectAlerts,
  AmountInputType,
} from "../../lib/api";
import { createProject, updateProject } from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";

interface Props {
  locale: Locale;
  project?: Project | null;
  redirectPath: string;
}

const defaultAlerts: ProjectAlerts = {
  warning_threshold: 60,
  critical_threshold: 30,
};

const emptyCostItem = (): CostItem => ({
  label: "",
  unit_price: 0,
  quantity: 1,
});

/** 既存 project から cost_items 初期値を生成 */
function initCostItems(project?: Project | null): CostItem[] {
  if (project?.cost_items && project.cost_items.length > 0) {
    return project.cost_items.map((ci) => ({
      label: ci.label,
      unit_price: ci.unit_price,
      quantity: ci.quantity,
    }));
  }
  return [emptyCostItem()];
}

function hasCostItems(items: CostItem[]): boolean {
  return items.some((ci) => ci.unit_price > 0);
}

export default function ProjectForm({ locale, project, redirectPath }: Props) {
  const isEdit = !!project;

  const [amountType, setAmountType] = useState<AmountInputType>(() => {
    if (!project) return "want";
    const hasWant =
      project.owner_want_monthly != null && project.owner_want_monthly > 0;
    const hasCost =
      (project.monthly_target ?? 0) > 0 || hasCostItems(initCostItems(project));
    if (hasWant && hasCost) return "both";
    if (hasCost) return "cost";
    return "want";
  });

  const [name, setName] = useState(project?.name ?? "");
  const [overview, setOverview] = useState(
    project?.overview ?? project?.description ?? "",
  );
  const [ownerWant, setOwnerWant] = useState(project?.owner_want_monthly ?? 0);
  const [costItems, setCostItems] = useState<CostItem[]>(() =>
    initCostItems(project),
  );

  // 期限
  const [deadlineType, setDeadlineType] = useState<"permanent" | "date">(
    project?.deadline ? "date" : "permanent",
  );
  const [deadlineValue, setDeadlineValue] = useState(
    project?.deadline ? project.deadline.slice(0, 10) : "",
  );

  // アラート閾値
  const [alertsEnabled, setAlertsEnabled] = useState(!!project?.alerts);
  const [alerts, setAlerts] = useState<ProjectAlerts>(
    project?.alerts ?? defaultAlerts,
  );

  // シェアメッセージ
  const [shareMessage, setShareMessage] = useState(
    project?.share_message ?? "",
  );

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // --- cost_items 操作 ---
  const updateCostItem = (idx: number, patch: Partial<CostItem>) => {
    setCostItems((prev) =>
      prev.map((ci, i) => (i === idx ? { ...ci, ...patch } : ci)),
    );
  };
  const addCostItem = () => {
    setCostItems((prev) => [...prev, emptyCostItem()]);
  };
  const removeCostItem = (idx: number) => {
    setCostItems((prev) => {
      const next = prev.filter((_, i) => i !== idx);
      return next.length === 0 ? [emptyCostItem()] : next;
    });
  };

  const doSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      // cost_items: ラベルか金額が入っている行のみ送信
      const validItems =
        amountType === "cost" || amountType === "both"
          ? costItems.filter(
              (ci) => ci.label.trim() !== "" || ci.unit_price > 0,
            )
          : null;

      const payload = {
        name,
        overview,
        share_message: shareMessage,
        deadline:
          deadlineType === "date" && deadlineValue ? deadlineValue : null,
        owner_want_monthly:
          amountType === "want" || amountType === "both"
            ? ownerWant > 0
              ? ownerWant
              : null
            : null,
        cost_items: validItems && validItems.length > 0 ? validItems : null,
        alerts: alertsEnabled ? alerts : null,
      };
      if (isEdit) {
        await updateProject(project!.id, payload);
        window.location.href = redirectPath;
      } else {
        const created = await createProject(payload);
        if (created.stripe_connect_url) {
          window.location.href = created.stripe_connect_url;
        } else {
          window.location.href = redirectPath;
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      onSubmit={doSubmit}
      className="card"
      style={{ maxWidth: "32rem", marginTop: "1.5rem" }}
    >
      {/* プロジェクト名 */}
      <div style={{ marginBottom: "1rem" }}>
        <label
          htmlFor="name"
          style={{ display: "block", marginBottom: "0.25rem" }}
        >
          プロジェクト名 *
        </label>
        <input
          id="name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          style={{ width: "100%", padding: "0.5rem" }}
        />
      </div>

      {/* 概要（Markdown） */}
      <div style={{ marginBottom: "1rem" }}>
        <label
          htmlFor="overview"
          style={{ display: "block", marginBottom: "0.25rem" }}
        >
          {t(locale, "projects.overviewLabel")}
        </label>
        <textarea
          id="overview"
          value={overview}
          onChange={(e) => setOverview(e.target.value)}
          rows={6}
          placeholder={t(locale, "projects.overviewPlaceholder")}
          style={{ width: "100%", padding: "0.5rem" }}
        />
        <small style={{ color: "var(--color-text-muted)" }}>
          {t(locale, "projects.overviewHint")}
        </small>
      </div>

      {/* シェアメッセージ */}
      <div style={{ marginBottom: "1rem" }}>
        <label
          htmlFor="shareMessage"
          style={{ display: "block", marginBottom: "0.25rem" }}
        >
          {t(locale, "share.formLabel")}
        </label>
        <textarea
          id="shareMessage"
          value={shareMessage}
          onChange={(e) => setShareMessage(e.target.value)}
          rows={2}
          placeholder={t(locale, "share.messagePlaceholder")}
          style={{ width: "100%", padding: "0.5rem" }}
        />
        <small style={{ color: "var(--color-text-muted)" }}>
          {t(locale, "share.formHint")}
        </small>
      </div>

      {/* 期限 */}
      <fieldset
        style={{
          marginBottom: "1rem",
          border: "1px solid var(--color-border)",
          padding: "1rem",
          borderRadius: "4px",
        }}
      >
        <legend>{t(locale, "projects.deadline")}</legend>
        <label style={{ display: "block", marginBottom: "0.5rem" }}>
          <input
            type="radio"
            name="deadlineType"
            checked={deadlineType === "permanent"}
            onChange={() => setDeadlineType("permanent")}
          />{" "}
          {t(locale, "projects.deadlinePermanent")}
        </label>
        <label style={{ display: "block" }}>
          <input
            type="radio"
            name="deadlineType"
            checked={deadlineType === "date"}
            onChange={() => setDeadlineType("date")}
          />{" "}
          {t(locale, "projects.deadlineDate")}
        </label>
        {deadlineType === "date" && (
          <input
            type="date"
            value={deadlineValue}
            onChange={(e) => setDeadlineValue(e.target.value)}
            style={{ marginTop: "0.5rem", padding: "0.5rem" }}
          />
        )}
      </fieldset>

      {/* 金額タイプ */}
      <fieldset
        style={{
          marginBottom: "1rem",
          border: "1px solid var(--color-border)",
          padding: "1rem",
          borderRadius: "4px",
        }}
      >
        <legend>{t(locale, "projects.amountType")}</legend>
        <label style={{ display: "block", marginBottom: "0.5rem" }}>
          <input
            type="radio"
            name="amountType"
            checked={amountType === "want"}
            onChange={() => setAmountType("want")}
          />{" "}
          {t(locale, "projects.amountTypeWant")}
        </label>
        <label style={{ display: "block", marginBottom: "0.5rem" }}>
          <input
            type="radio"
            name="amountType"
            checked={amountType === "cost"}
            onChange={() => setAmountType("cost")}
          />{" "}
          {t(locale, "projects.amountTypeCost")}
        </label>
        <label style={{ display: "block" }}>
          <input
            type="radio"
            name="amountType"
            checked={amountType === "both"}
            onChange={() => setAmountType("both")}
          />{" "}
          {t(locale, "projects.amountTypeBoth")}
        </label>
      </fieldset>

      {/* 最低希望額 */}
      {(amountType === "want" || amountType === "both") && (
        <div style={{ marginBottom: "1rem" }}>
          <label
            htmlFor="ownerWant"
            style={{ display: "block", marginBottom: "0.25rem" }}
          >
            {t(locale, "projects.ownerWant")} (¥)
          </label>
          <input
            id="ownerWant"
            type="number"
            min={0}
            value={ownerWant || ""}
            onChange={(e) => setOwnerWant(parseInt(e.target.value, 10) || 0)}
            style={{ width: "100%", padding: "0.5rem" }}
          />
        </div>
      )}

      {/* 見積詳細（単価×数量） */}
      {(amountType === "cost" || amountType === "both") && (
        <div
          style={{
            marginBottom: "1rem",
            padding: "1rem",
            border: "1px solid var(--color-border)",
            borderRadius: "4px",
          }}
        >
          <h3 style={{ marginTop: 0 }}>
            {t(locale, "projects.costBreakdown")}
          </h3>
          {costItems.map((ci, idx) => {
            const subtotal = (ci.unit_price || 0) * (ci.quantity || 0);
            return (
              <div
                key={idx}
                style={{
                  display: "flex",
                  gap: "0.5rem",
                  alignItems: "flex-end",
                  marginBottom: "0.5rem",
                }}
              >
                <div style={{ flex: 2 }}>
                  {idx === 0 && (
                    <label
                      style={{
                        display: "block",
                        marginBottom: "0.25rem",
                        fontSize: "0.85rem",
                      }}
                    >
                      {t(locale, "projects.costItemLabel")}
                    </label>
                  )}
                  <input
                    type="text"
                    value={ci.label}
                    onChange={(e) =>
                      updateCostItem(idx, { label: e.target.value })
                    }
                    placeholder={t(locale, "projects.costItemLabelPlaceholder")}
                    style={{ width: "100%", padding: "0.5rem" }}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  {idx === 0 && (
                    <label
                      style={{
                        display: "block",
                        marginBottom: "0.25rem",
                        fontSize: "0.85rem",
                      }}
                    >
                      {t(locale, "projects.costItemUnitPrice")}
                    </label>
                  )}
                  <input
                    type="number"
                    min={0}
                    value={ci.unit_price || ""}
                    onChange={(e) =>
                      updateCostItem(idx, {
                        unit_price: parseInt(e.target.value, 10) || 0,
                      })
                    }
                    style={{ width: "100%", padding: "0.5rem" }}
                  />
                </div>
                <div style={{ width: "4.5rem" }}>
                  {idx === 0 && (
                    <label
                      style={{
                        display: "block",
                        marginBottom: "0.25rem",
                        fontSize: "0.85rem",
                      }}
                    >
                      {t(locale, "projects.costItemQuantity")}
                    </label>
                  )}
                  <input
                    type="number"
                    min={1}
                    value={ci.quantity || ""}
                    onChange={(e) =>
                      updateCostItem(idx, {
                        quantity: parseInt(e.target.value, 10) || 1,
                      })
                    }
                    style={{ width: "100%", padding: "0.5rem" }}
                  />
                </div>
                <div
                  style={{
                    width: "5.5rem",
                    textAlign: "right",
                    paddingBottom: "0.5rem",
                    fontSize: "0.9rem",
                    color: "var(--color-text-muted)",
                  }}
                >
                  {idx === 0 && (
                    <label
                      style={{
                        display: "block",
                        marginBottom: "0.25rem",
                        fontSize: "0.85rem",
                      }}
                    >
                      {t(locale, "projects.costItemSubtotal")}
                    </label>
                  )}
                  ¥{subtotal.toLocaleString()}
                </div>
                <button
                  type="button"
                  onClick={() => removeCostItem(idx)}
                  style={{
                    padding: "0.5rem",
                    background: "none",
                    border: "1px solid var(--color-border)",
                    borderRadius: "4px",
                    cursor: "pointer",
                    lineHeight: 1,
                  }}
                  title="削除"
                >
                  ✕
                </button>
              </div>
            );
          })}
          <button
            type="button"
            onClick={addCostItem}
            style={{
              marginTop: "0.25rem",
              padding: "0.4rem 0.75rem",
              background: "none",
              border: "1px dashed var(--color-border)",
              borderRadius: "4px",
              cursor: "pointer",
              fontSize: "0.9rem",
            }}
          >
            {t(locale, "projects.costItemAddRow")}
          </button>
          {/* 見積合計 */}
          <div
            style={{
              marginTop: "0.75rem",
              paddingTop: "0.75rem",
              borderTop: "1px solid var(--color-border)",
              display: "flex",
              justifyContent: "flex-end",
              alignItems: "center",
              gap: "0.5rem",
              fontWeight: "bold",
            }}
          >
            <span>{t(locale, "projects.costTotal")}</span>
            <span style={{ fontSize: "1.1rem" }}>
              ¥
              {costItems
                .reduce(
                  (sum, ci) => sum + (ci.unit_price || 0) * (ci.quantity || 0),
                  0,
                )
                .toLocaleString()}
              <span style={{ fontWeight: "normal", fontSize: "0.85rem" }}>
                /月
              </span>
            </span>
          </div>
        </div>
      )}

      {/* アラート閾値 */}
      <fieldset
        style={{
          marginBottom: "1rem",
          border: "1px solid var(--color-border)",
          padding: "1rem",
          borderRadius: "4px",
        }}
      >
        <legend>{t(locale, "projects.alertsTitle")}</legend>
        <label
          style={{
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
            marginBottom: "0.75rem",
            cursor: "pointer",
          }}
        >
          <input
            type="checkbox"
            checked={alertsEnabled}
            onChange={(e) => setAlertsEnabled(e.target.checked)}
          />
          アラートを設定する
        </label>
        {alertsEnabled && (
          <>
            <p
              style={{
                margin: "0 0 0.75rem",
                fontSize: "0.85rem",
                color: "var(--color-text-muted)",
              }}
            >
              {t(locale, "projects.alertsHint")}
            </p>
            <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
              <div style={{ flex: 1, minWidth: "120px" }}>
                <label
                  htmlFor="warningThreshold"
                  style={{
                    display: "block",
                    marginBottom: "0.25rem",
                    fontSize: "0.9rem",
                  }}
                >
                  {t(locale, "projects.warningThreshold")}
                </label>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "0.25rem",
                  }}
                >
                  <input
                    id="warningThreshold"
                    type="number"
                    min={1}
                    max={100}
                    value={alerts.warning_threshold || ""}
                    onChange={(e) =>
                      setAlerts({
                        ...alerts,
                        warning_threshold: parseInt(e.target.value, 10) || 0,
                      })
                    }
                    style={{ width: "80px", padding: "0.5rem" }}
                  />
                  <span>%</span>
                </div>
              </div>
              <div style={{ flex: 1, minWidth: "120px" }}>
                <label
                  htmlFor="criticalThreshold"
                  style={{
                    display: "block",
                    marginBottom: "0.25rem",
                    fontSize: "0.9rem",
                  }}
                >
                  {t(locale, "projects.criticalThreshold")}
                </label>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "0.25rem",
                  }}
                >
                  <input
                    id="criticalThreshold"
                    type="number"
                    min={1}
                    max={100}
                    value={alerts.critical_threshold || ""}
                    onChange={(e) =>
                      setAlerts({
                        ...alerts,
                        critical_threshold: parseInt(e.target.value, 10) || 0,
                      })
                    }
                    style={{ width: "80px", padding: "0.5rem" }}
                  />
                  <span>%</span>
                </div>
              </div>
            </div>
          </>
        )}
      </fieldset>

      {/* Stripe Connect 案内（新規作成のみ） */}
      {!isEdit && (
        <div
          style={{
            marginBottom: "1.5rem",
            padding: "1rem",
            border: "1px solid var(--color-border)",
            borderRadius: "4px",
          }}
        >
          <h3 style={{ marginTop: 0 }}>
            {t(locale, "projects.stripeConnectTitle")}
          </h3>
          <p style={{ marginBottom: "0.5rem", fontSize: "0.9rem" }}>
            {t(locale, "projects.stripeConnectDesc")}
          </p>
          <p
            style={{
              marginBottom: 0,
              fontSize: "0.8rem",
              color: "var(--color-text-muted)",
            }}
          >
            {t(locale, "projects.stripeConnectNote")}
          </p>
        </div>
      )}

      {error && (
        <p style={{ color: "var(--color-danger)", marginBottom: "1rem" }}>
          {error}
        </p>
      )}

      <button type="submit" className="btn btn-accent" disabled={submitting}>
        {submitting
          ? t(locale, "projects.loading")
          : isEdit
            ? t(locale, "projects.editProject")
            : t(locale, "projects.newProject")}
      </button>
    </form>
  );
}
