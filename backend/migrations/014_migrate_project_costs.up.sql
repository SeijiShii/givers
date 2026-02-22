-- project_costs → project_cost_items へデータ移行
INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
SELECT project_id, 'サーバー費用', 'monthly', server_cost_monthly, 0, 0, 0
FROM project_costs
WHERE server_cost_monthly > 0;

INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
SELECT project_id, '開発者費用', 'daily_x_days', 0, dev_cost_per_day, dev_days_per_month, 1
FROM project_costs
WHERE dev_cost_per_day > 0 OR dev_days_per_month > 0;

INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
SELECT project_id, 'その他費用', 'monthly', other_cost_monthly, 0, 0, 2
FROM project_costs
WHERE other_cost_monthly > 0;

-- projects.monthly_target を集計して更新
UPDATE projects p SET monthly_target = (
    SELECT COALESCE(SUM(
        CASE unit_type
            WHEN 'monthly'      THEN amount_monthly
            WHEN 'daily_x_days' THEN rate_per_day * days_per_month
        END
    ), 0)
    FROM project_cost_items
    WHERE project_id = p.id
);

DROP TABLE IF EXISTS project_costs;
