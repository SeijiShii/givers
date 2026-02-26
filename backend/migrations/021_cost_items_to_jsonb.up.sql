-- projects に cost_items JSONB カラム追加
ALTER TABLE projects ADD COLUMN IF NOT EXISTS cost_items JSONB;

-- 既存データ移行: project_cost_items → JSONB
UPDATE projects p SET cost_items = sub.items
FROM (
  SELECT ci.project_id, jsonb_agg(jsonb_build_object(
    'label', ci.label,
    'unit_price', CASE ci.unit_type
      WHEN 'daily_x_days' THEN ci.rate_per_day
      ELSE ci.amount_monthly END,
    'quantity', CASE ci.unit_type
      WHEN 'daily_x_days' THEN ci.days_per_month
      ELSE 1 END
  ) ORDER BY ci.sort_order) AS items
  FROM project_cost_items ci
  GROUP BY ci.project_id
) sub
WHERE sub.project_id = p.id;

-- monthly_target 再計算
UPDATE projects SET monthly_target = COALESCE((
  SELECT SUM((item->>'unit_price')::int * (item->>'quantity')::int)
  FROM jsonb_array_elements(cost_items) item
), 0)
WHERE cost_items IS NOT NULL;

-- 旧テーブル削除
DROP TABLE IF EXISTS project_cost_items;
