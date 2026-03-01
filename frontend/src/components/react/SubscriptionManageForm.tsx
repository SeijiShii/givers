import { useState } from "react";
import { t, type Locale } from "../../lib/i18n";
import {
  updateRecurringDonation,
  pauseRecurringDonation,
  resumeRecurringDonation,
  deleteRecurringDonation,
  type RecurringDonation,
} from "../../lib/api";
import ConfirmDialog from "./ConfirmDialog";

interface Props {
  locale: Locale;
  donation: RecurringDonation;
  onUpdate: (updated: RecurringDonation) => void;
  onCancel: () => void;
}

export default function SubscriptionManageForm({
  locale,
  donation,
  onUpdate,
  onCancel,
}: Props) {
  const [editingAmount, setEditingAmount] = useState(false);
  const [amount, setAmount] = useState(donation.amount);
  const [saving, setSaving] = useState(false);
  const [nextMessage, setNextMessage] = useState(
    donation.next_billing_message ?? "",
  );
  const [messageSaved, setMessageSaved] = useState(false);
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);

  const isPaused = donation.status === "paused";

  const handleSaveAmount = async () => {
    if (amount <= 0 || amount === donation.amount) return;
    setSaving(true);
    try {
      const updated = await updateRecurringDonation(donation.id, { amount });
      onUpdate(updated);
      setEditingAmount(false);
    } finally {
      setSaving(false);
    }
  };

  const handlePauseResume = async () => {
    setSaving(true);
    try {
      if (isPaused) {
        await resumeRecurringDonation(donation.id);
      } else {
        await pauseRecurringDonation(donation.id);
      }
      onUpdate({
        ...donation,
        status: isPaused ? "active" : "paused",
      });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    setShowCancelConfirm(false);
    setSaving(true);
    try {
      await deleteRecurringDonation(donation.id);
      onCancel();
    } finally {
      setSaving(false);
    }
  };

  const handleSaveMessage = async () => {
    setSaving(true);
    try {
      const updated = await updateRecurringDonation(donation.id, {
        next_billing_message: nextMessage,
      });
      onUpdate(updated);
      setMessageSaved(true);
      setTimeout(() => setMessageSaved(false), 2000);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={{ marginTop: "1rem" }}>
      <ConfirmDialog
        open={showCancelConfirm}
        title={t(locale, "projects.subscriptionCancelConfirmTitle")}
        message={t(locale, "projects.subscriptionCancelConfirm")}
        confirmLabel={t(locale, "projects.subscriptionCancel")}
        cancelLabel={t(locale, "projects.cancel")}
        danger
        onConfirm={handleDelete}
        onCancel={() => setShowCancelConfirm(false)}
      />

      <h3 style={{ marginTop: 0, marginBottom: "0.75rem" }}>
        {t(locale, "projects.subscriptionManageTitle")}
      </h3>

      {isPaused && (
        <div
          style={{
            padding: "0.4rem 0.75rem",
            marginBottom: "1rem",
            borderRadius: "6px",
            background: "var(--color-warning-muted, rgba(200,150,0,0.1))",
            fontSize: "0.9rem",
          }}
        >
          {t(locale, "projects.subscriptionPaused")}
        </div>
      )}

      {/* Current amount / Edit amount */}
      <div style={{ marginBottom: "1rem" }}>
        <p
          style={{
            margin: "0 0 0.5rem",
            fontSize: "0.95rem",
            fontWeight: 500,
          }}
        >
          {t(locale, "projects.subscriptionCurrentAmount")}
        </p>
        {editingAmount ? (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              flexWrap: "wrap",
            }}
          >
            <input
              type="number"
              min={100}
              step={100}
              value={amount}
              onChange={(e) => setAmount(Number(e.target.value) || 0)}
              style={{ width: "7rem", padding: "0.4rem 0.5rem" }}
            />
            <span>円/月</span>
            <button
              type="button"
              className="btn btn-primary"
              style={{ fontSize: "0.85rem" }}
              onClick={handleSaveAmount}
              disabled={saving || amount <= 0 || amount === donation.amount}
            >
              {saving ? "..." : t(locale, "projects.save")}
            </button>
            <button
              type="button"
              className="btn"
              style={{ fontSize: "0.85rem" }}
              onClick={() => {
                setEditingAmount(false);
                setAmount(donation.amount);
              }}
            >
              {t(locale, "projects.cancel")}
            </button>
          </div>
        ) : (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.75rem",
            }}
          >
            <span style={{ fontSize: "1.1rem", fontWeight: 600 }}>
              ¥{donation.amount.toLocaleString()}/月
            </span>
            <button
              type="button"
              className="btn"
              style={{ fontSize: "0.85rem" }}
              onClick={() => setEditingAmount(true)}
              disabled={saving}
            >
              {t(locale, "projects.subscriptionChangeAmount")}
            </button>
          </div>
        )}
      </div>

      {/* Next billing message */}
      <div style={{ marginBottom: "1rem" }}>
        <label
          htmlFor="next-billing-msg"
          style={{
            display: "block",
            marginBottom: "0.25rem",
            fontSize: "0.95rem",
            fontWeight: 500,
          }}
        >
          {t(locale, "projects.subscriptionNextMessage")}
        </label>
        <textarea
          id="next-billing-msg"
          value={nextMessage}
          onChange={(e) => {
            setNextMessage(e.target.value);
            setMessageSaved(false);
          }}
          placeholder={t(
            locale,
            "projects.subscriptionNextMessagePlaceholder",
          )}
          rows={2}
          style={{
            width: "100%",
            padding: "0.5rem",
            border: "1px solid var(--color-border)",
            borderRadius: "6px",
            fontFamily: "inherit",
          }}
        />
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
            marginTop: "0.25rem",
          }}
        >
          <button
            type="button"
            className="btn"
            style={{ fontSize: "0.85rem" }}
            onClick={handleSaveMessage}
            disabled={
              saving || nextMessage === (donation.next_billing_message ?? "")
            }
          >
            {t(locale, "projects.save")}
          </button>
          {messageSaved && (
            <span
              style={{
                fontSize: "0.85rem",
                color: "var(--color-primary)",
              }}
            >
              {t(locale, "projects.subscriptionNextMessageSaved")}
            </span>
          )}
        </div>
      </div>

      {/* Action buttons */}
      <div
        style={{
          display: "flex",
          gap: "0.5rem",
          flexWrap: "wrap",
          marginTop: "1rem",
        }}
      >
        <button
          type="button"
          className="btn"
          onClick={handlePauseResume}
          disabled={saving}
        >
          {isPaused
            ? t(locale, "projects.subscriptionResume")
            : t(locale, "projects.subscriptionPause")}
        </button>
        <button
          type="button"
          className="btn"
          style={{ color: "var(--color-danger)" }}
          onClick={() => setShowCancelConfirm(true)}
          disabled={saving}
        >
          {t(locale, "projects.subscriptionCancel")}
        </button>
      </div>
    </div>
  );
}
