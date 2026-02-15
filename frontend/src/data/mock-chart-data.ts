/** プロジェクトチャート用のモックデータ
 * 最低金額・目標金額は変更することがあるため、非連続階段状（ステップ）で表示
 * 各月の達成額は折れ線グラフ */

export interface ChartDataPoint {
  month: string; // "2025-01" 形式
  /** 最低希望額（月額）※変更時のみ値が変わる */
  minAmount: number;
  /** 目標金額（必要額、月額）※変更時のみ値が変わる */
  targetAmount: number;
  /** その月の実際の寄付額 */
  actualAmount: number;
}

export interface ProjectChartData {
  labels: string[];
  data: ChartDataPoint[];
}

const now = new Date();
const monthStr = (y: number, m: number) => `${y}-${String(m).padStart(2, '0')}`;

/** 直近Nヶ月のデータを生成 */
function generateMonths(n: number): string[] {
  const result: string[] = [];
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    result.push(monthStr(d.getFullYear(), d.getMonth() + 1));
  }
  return result;
}

/** プロジェクトID別のチャートモックデータ */
export const MOCK_CHART_DATA: Record<string, ChartDataPoint[]> = {
  'mock-1': (() => {
    const months = generateMonths(6);
    const baseMin = 35000;
    const baseTarget = 50000;
    const actuals = [28000, 32000, 35000, 42000, 45000, 42000]; // 最後が現在
    return months.map((month, i) => ({
      month,
      minAmount: i >= 4 ? 55000 : baseMin, // 2ヶ月前に最低額変更
      targetAmount: i >= 3 ? 60000 : baseTarget, // 3ヶ月前に目標変更（目標 > 最低）
      actualAmount: actuals[i],
    }));
  })(),
  'mock-2': (() => {
    const months = generateMonths(6);
    return months.map((month, i) => ({
      month,
      minAmount: 56000,
      targetAmount: 80000,
      actualAmount: [60000, 72000, 85000, 90000, 92000, 95000][i],
    }));
  })(),
  'mock-3': (() => {
    const months = generateMonths(6);
    return months.map((month, i) => ({
      month,
      minAmount: 25000,
      targetAmount: 30000,
      actualAmount: [5000, 6000, 7000, 7500, 7800, 8000][i],
    }));
  })(),
  'mock-4': (() => {
    const months = generateMonths(6);
    return months.map((month, i) => ({
      month,
      minAmount: i >= 2 ? 100000 : 75000,
      targetAmount: 120000,
      actualAmount: [40000, 45000, 48000, 50000, 51000, 52000][i],
    }));
  })(),
  'mock-5': (() => {
    const months = generateMonths(6);
    return months.map((month, i) => ({
      month,
      minAmount: 28000,
      targetAmount: 40000,
      actualAmount: [0, 5000, 12000, 15000, 18000, 20000][i],
    }));
  })(),
  'mock-6': (() => {
    const months = generateMonths(6);
    return months.map((month, i) => ({
      month,
      minAmount: 15000,
      targetAmount: 25000,
      actualAmount: [10000, 15000, 18000, 20000, 21000, 22000][i],
    }));
  })(),
};
