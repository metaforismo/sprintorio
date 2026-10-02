ALTER TABLE projects
    ADD COLUMN delivery_plan JSONB NOT NULL DEFAULT '{"product_name":"","objective":"","success_metric":"","target_release":"","milestones":[],"test_cases":[]}'::jsonb,
    ADD COLUMN delivery_plan_version INTEGER NOT NULL DEFAULT 0 CHECK (delivery_plan_version >= 0);
