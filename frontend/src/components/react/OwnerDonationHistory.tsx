import { useEffect, useState } from "react";
import type { OwnerDonationItem } from "../../lib/api";
import { getOwnerDonations, refundDonationByOwner } from "../../lib/api";
import { t, type Locale } from "../../lib/i18n";
import ConfirmDialog from "./ConfirmDialog";

const PAGE_SIZE = 50;

interface Props {
  projectId: string;
  locale: Locale;
}

function formatDate(dateStr: string, locale: Locale): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString(locale === "ja" ? "ja-JP" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function OwnerDonationHistory({ projectId, locale }: Props) {
  const [donations, setDonations] = useState<OwnerDonationItem[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [sourceFilter, setSourceFilter] = useState("");
  const [donorFilter, setDonorFilter] = useState("");
  const [hasMessageFilter, setHasMessageFilter] = useState(false);
  const [sortOrder, setSortOrder] = useState<"desc" | "asc">("desc");
  const [refundConfirmId, setRefundConfirmId] = useState<string | null>(null);
  const [refundingId, setRefundingId] = useState<string | null>(null);
  const [refundError, setRefundError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getOwnerDonations(projectId, {
      limit: PAGE_SIZE,
      offset,
      sort: sortOrder,
      source: sourceFilter || undefined,
      donor: donorFilter || undefined,
      has_message: hasMessageFilter || undefined,
    })
      .then((res) => {
        if (cancelled) return;
        setDonations(res.donations ?? []);
        setTotal(res.total);
      })
      .catch(() => {
        if (!cancelled) {
          setDonations([]);
          setTotal(0);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    projectId,
    offset,
    sortOrder,
    sourceFilter,
    donorFilter,
    hasMessageFilter,
  ]);

  // Reset offset when filters change
  useEffect(() => {
    setOffset(0);
  }, [sourceFilter, donorFilter, hasMessageFilter]);

  const handleRefund = async (donationId: string) => {
    setRefundConfirmId(null);
    setRefundingId(donationId);
    setRefundError(null);
    try {
      await refundDonationByOwner(projectId, donationId);
      // Update local state to reflect refund
      setDonations((prev) =>
        prev.map((d) =>
          d.id === donationId ? { ...d, refund_status: "completed" } : d,
        ),
      );
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "";
      if (msg.includes("409")) {
        setRefundError(
          t(locale, "projects.donationHistoryRefundAlreadyRefunded"),
        );
      } else {
        setRefundError(t(locale, "projects.donationHistoryRefundError"));
      }
    } finally {
      setRefundingId(null);
    }
  };

  const refundConfirmDonation = donations.find((d) => d.id === refundConfirmId);

  return (
    <div className="card" style={{ padding: "1.5rem" }}>
      <h3 style={{ marginTop: 0, marginBottom: "1rem" }}>
        {t(locale, "projects.donationHistoryTitle")}
      </h3>

      {/* Filters */}
      <div
        style={{
          marginBottom: "1rem",
          display: "flex",
          gap: "0.75rem",
          flexWrap: "wrap",
          alignItems: "center",
        }}
      >
        <select
          value={sourceFilter}
          onChange={(e) => setSourceFilter(e.target.value)}
          style={{ padding: "0.5rem" }}
        >
          <option value="">
            {t(locale, "projects.donationHistorySourceAll")}
          </option>
          <option value="checkout">
            {t(locale, "projects.donationHistorySourceCheckout")}
          </option>
          <option value="subscription_renewal">
            {t(locale, "projects.donationHistorySourceRenewal")}
          </option>
        </select>

        <input
          type="text"
          value={donorFilter}
          onChange={(e) => setDonorFilter(e.target.value)}
          placeholder={t(locale, "projects.donationHistoryFilterDonor")}
          style={{ padding: "0.5rem", maxWidth: "200px" }}
        />

        <label
          style={{
            display: "flex",
            alignItems: "center",
            gap: "0.25rem",
            fontSize: "0.9rem",
            cursor: "pointer",
          }}
        >
          <input
            type="checkbox"
            checked={hasMessageFilter}
            onChange={(e) => setHasMessageFilter(e.target.checked)}
          />
          {t(locale, "projects.donationHistoryHasMessage")}
        </label>

        <button
          type="button"
          className="btn btn-small"
          onClick={() => setSortOrder(sortOrder === "desc" ? "asc" : "desc")}
          style={{ padding: "0.5rem 0.75rem", fontSize: "0.85rem" }}
        >
          {sortOrder === "desc"
            ? t(locale, "projects.donationHistorySortNewFirst")
            : t(locale, "projects.donationHistorySortOldFirst")}
        </button>
      </div>

      {loading ? (
        <p style={{ color: "var(--color-text-muted)" }}>
          {t(locale, "projects.loading")}
        </p>
      ) : donations.length === 0 ? (
        <p style={{ color: "var(--color-text-muted)" }}>
          {t(locale, "projects.donationHistoryEmpty")}
        </p>
      ) : (
        <>
          <div style={{ overflowX: "auto" }}>
            <table
              style={{
                width: "100%",
                borderCollapse: "collapse",
                fontSize: "0.9rem",
              }}
            >
              <thead>
                <tr
                  style={{
                    borderBottom: "2px solid var(--color-border)",
                    textAlign: "left",
                  }}
                >
                  <th style={{ padding: "0.5rem" }}>
                    {t(locale, "projects.donationHistoryDonor")}
                  </th>
                  <th style={{ padding: "0.5rem" }}>
                    {t(locale, "projects.donationHistoryAmount")}
                  </th>
                  <th style={{ padding: "0.5rem" }}>
                    {t(locale, "projects.donationHistorySource")}
                  </th>
                  <th style={{ padding: "0.5rem" }}>
                    {t(locale, "projects.donationHistoryMessage")}
                  </th>
                  <th style={{ padding: "0.5rem" }}>
                    {t(locale, "projects.donationHistoryDate")}
                  </th>
                  <th style={{ padding: "0.5rem" }}>
                    {t(locale, "projects.donationHistoryStatus")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {donations.map((d) => (
                  <tr
                    key={d.id}
                    style={{
                      borderBottom: "1px solid var(--color-border-light)",
                    }}
                  >
                    <td style={{ padding: "0.5rem" }}>
                      {d.donor_name ||
                        t(locale, "projects.donationHistoryAnonymous")}
                      {d.is_recurring && (
                        <span
                          style={{
                            marginLeft: "0.25rem",
                            fontSize: "0.75rem",
                            color: "var(--color-text-muted)",
                          }}
                        >
                          ({t(locale, "projects.donationHistoryRecurring")})
                        </span>
                      )}
                    </td>
                    <td style={{ padding: "0.5rem" }}>
                      ¥{d.amount.toLocaleString()}
                    </td>
                    <td style={{ padding: "0.5rem", fontSize: "0.85rem" }}>
                      {d.source === "subscription_renewal"
                        ? t(locale, "projects.donationHistorySourceRenewal")
                        : t(locale, "projects.donationHistorySourceCheckout")}
                    </td>
                    <td
                      style={{
                        padding: "0.5rem",
                        whiteSpace: "pre-wrap",
                        maxWidth: "250px",
                      }}
                    >
                      {d.message || "—"}
                    </td>
                    <td
                      style={{
                        padding: "0.5rem",
                        fontSize: "0.85rem",
                        color: "var(--color-text-muted)",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {formatDate(d.created_at, locale)}
                    </td>
                    <td style={{ padding: "0.5rem", whiteSpace: "nowrap" }}>
                      {d.refund_status === "completed" ? (
                        <span
                          style={{
                            color: "var(--color-text-muted)",
                            fontSize: "0.85rem",
                          }}
                        >
                          {t(locale, "projects.donationHistoryRefundCompleted")}
                        </span>
                      ) : d.refund_status === "pending" ||
                        refundingId === d.id ? (
                        <span
                          style={{
                            color: "var(--color-text-muted)",
                            fontSize: "0.85rem",
                          }}
                        >
                          {t(locale, "projects.donationHistoryRefundPending")}
                        </span>
                      ) : (
                        <button
                          type="button"
                          className="btn btn-small"
                          style={{
                            fontSize: "0.8rem",
                            padding: "0.25rem 0.5rem",
                          }}
                          onClick={() => setRefundConfirmId(d.id)}
                        >
                          {t(locale, "projects.donationHistoryRefundBtn")}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {total > PAGE_SIZE && (
            <div
              style={{
                marginTop: "1rem",
                display: "flex",
                gap: "0.5rem",
                justifyContent: "center",
                alignItems: "center",
              }}
            >
              <button
                type="button"
                className="btn"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
              >
                &laquo;
              </button>
              <span style={{ padding: "0.5rem", fontSize: "0.85rem" }}>
                {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} / {total}
              </span>
              <button
                type="button"
                className="btn"
                disabled={offset + PAGE_SIZE >= total}
                onClick={() => setOffset(offset + PAGE_SIZE)}
              >
                &raquo;
              </button>
            </div>
          )}
        </>
      )}

      {refundError && (
        <p
          style={{
            marginTop: "0.75rem",
            color: "var(--color-danger)",
            fontSize: "0.9rem",
          }}
        >
          {refundError}
        </p>
      )}

      <ConfirmDialog
        open={refundConfirmId !== null}
        title={t(locale, "projects.donationHistoryRefund")}
        message={
          refundConfirmDonation?.is_recurring
            ? t(locale, "projects.donationHistoryRefundConfirmSubscription")
            : t(locale, "projects.donationHistoryRefundConfirm")
        }
        confirmLabel={t(locale, "projects.donationHistoryRefundBtn")}
        cancelLabel={t(locale, "projects.cancel")}
        danger
        onConfirm={() => {
          if (refundConfirmId) handleRefund(refundConfirmId);
        }}
        onCancel={() => {
          setRefundConfirmId(null);
          setRefundError(null);
        }}
      />
    </div>
  );
}
