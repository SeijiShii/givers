import { useEffect, useState } from 'react';
import {
  ResponsiveContainer,
  ComposedChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from 'recharts';
import { getProjectChart, type ChartDataPoint } from '../../../lib/api';

interface Props {
  projectId: string;
  minAmountLabel: string;
  targetAmountLabel: string;
  actualAmountLabel: string;
  noDataLabel: string;
}

const COLORS = {
  min: '#d9534f', // 赤に近いオレンジ
  target: '#4a7bc8', // 青系
  actual: 'var(--color-primary-light)', // 緑のまま
};

function formatYAxis(value: number): string {
  if (value >= 10000) return `¥${(value / 10000).toFixed(0)}万`;
  return `¥${value.toLocaleString()}`;
}

function formatTooltip(value: number): string {
  return `¥${value.toLocaleString()}`;
}

export default function ProjectChart({
  projectId,
  minAmountLabel,
  targetAmountLabel,
  actualAmountLabel,
  noDataLabel,
}: Props) {
  const [data, setData] = useState<ChartDataPoint[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getProjectChart(projectId)
      .then(setData)
      .catch(() => setData([]))
      .finally(() => setLoading(false));
  }, [projectId]);

  if (loading) {
    return <p style={{ color: 'var(--color-text-muted)', fontSize: '0.9rem' }}>読み込み中...</p>;
  }

  if (data.length === 0) {
    return <p style={{ color: 'var(--color-text-muted)', fontSize: '0.9rem' }}>{noDataLabel}</p>;
  }

  return (
    <div style={{ width: '100%', height: 280, marginTop: '1rem' }}>
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 8 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
          <XAxis
            dataKey="month"
            tick={{ fontSize: 11, fill: 'var(--color-text-muted)' }}
            tickFormatter={(v: string) => v.replace('-', '/')}
          />
          <YAxis
            tick={{ fontSize: 11, fill: 'var(--color-text-muted)' }}
            tickFormatter={formatYAxis}
          />
          <Tooltip
            formatter={(value: unknown, name: string) => {
              const label =
                name === 'minAmount'
                  ? minAmountLabel
                  : name === 'targetAmount'
                    ? targetAmountLabel
                    : actualAmountLabel;
              return [formatTooltip(Number(value)), label];
            }}
            labelFormatter={(label: string) => label.replace('-', '/')}
            contentStyle={{
              backgroundColor: 'var(--color-bg-card)',
              border: '1px solid var(--color-border)',
              borderRadius: '6px',
            }}
          />
          <Legend
            wrapperStyle={{ fontSize: '0.85rem' }}
            formatter={(value) => {
              if (value === 'minAmount') return minAmountLabel;
              if (value === 'targetAmount') return targetAmountLabel;
              if (value === 'actualAmount') return actualAmountLabel;
              return value;
            }}
          />
          {/* 最低金額: 階段状（変更時のみ値が変わる） */}
          <Line
            type="stepAfter"
            dataKey="minAmount"
            name="minAmount"
            stroke={COLORS.min}
            strokeWidth={2}
            dot={{ r: 3, fill: COLORS.min }}
            connectNulls
          />
          {/* 目標金額: 階段状 */}
          <Line
            type="stepAfter"
            dataKey="targetAmount"
            name="targetAmount"
            stroke={COLORS.target}
            strokeWidth={2}
            dot={{ r: 3, fill: COLORS.target }}
            connectNulls
          />
          {/* 各月の達成額: 折れ線 */}
          <Line
            type="monotone"
            dataKey="actualAmount"
            name="actualAmount"
            stroke={COLORS.actual}
            strokeWidth={2}
            dot={{ r: 4, fill: COLORS.actual }}
            connectNulls
          />
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
