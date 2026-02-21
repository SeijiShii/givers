CREATE TABLE IF NOT EXISTS platform_health (
    id INT PRIMARY KEY DEFAULT 1,
    monthly_cost INTEGER NOT NULL DEFAULT 0,
    current_monthly INTEGER NOT NULL DEFAULT 0,
    warning_threshold INTEGER NOT NULL DEFAULT 60,
    critical_threshold INTEGER NOT NULL DEFAULT 30,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (id = 1)
);

-- Insert the singleton row with default values
INSERT INTO platform_health (id) VALUES (1) ON CONFLICT DO NOTHING;
