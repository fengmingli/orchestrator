-- 移除外键约束脚本
-- 重新创建表以移除外键约束

.echo on
.headers on

SELECT '=== 开始移除外键约束 ===' as info;

-- 备份数据
SELECT '=== 备份数据 ===' as info;

-- 备份 workflow_template_steps 数据
CREATE TABLE workflow_template_steps_backup AS SELECT * FROM workflow_template_steps;
SELECT 'workflow_template_steps 数据已备份，条数: ' || COUNT(*) FROM workflow_template_steps_backup;

-- 备份 workflow_step_executions 数据
CREATE TABLE workflow_step_executions_backup AS SELECT * FROM workflow_step_executions;
SELECT 'workflow_step_executions 数据已备份，条数: ' || COUNT(*) FROM workflow_step_executions_backup;

-- 删除原表
SELECT '=== 删除原表 ===' as info;
DROP TABLE workflow_template_steps;
DROP TABLE workflow_step_executions;

-- 重新创建表（无外键约束）
SELECT '=== 重新创建表（无外键约束） ===' as info;

CREATE TABLE `workflow_template_steps` (
    `id` char(36) PRIMARY KEY,
    `template_id` char(36) NOT NULL,
    `step_id` char(36) NOT NULL,
    `dependencies` text,
    `run_mode` text DEFAULT "serial",
    `on_failure` text DEFAULT "abort",
    `order` integer,
    `created_at` datetime
);

CREATE TABLE `workflow_step_executions` (
    `id` char(36) PRIMARY KEY,
    `execution_id` char(36) NOT NULL,
    `step_id` char(36) NOT NULL,
    `status` text DEFAULT "pending",
    `output` text,
    `error` text,
    `started_at` datetime,
    `finished_at` datetime,
    `duration` integer,
    `retry_count` integer DEFAULT 0,
    `created_at` datetime,
    `updated_at` datetime
);

-- 恢复数据
SELECT '=== 恢复数据 ===' as info;

INSERT INTO workflow_template_steps SELECT * FROM workflow_template_steps_backup;
SELECT 'workflow_template_steps 数据已恢复，条数: ' || COUNT(*) FROM workflow_template_steps;

INSERT INTO workflow_step_executions SELECT * FROM workflow_step_executions_backup;
SELECT 'workflow_step_executions 数据已恢复，条数: ' || COUNT(*) FROM workflow_step_executions;

-- 删除备份表
SELECT '=== 清理备份表 ===' as info;
DROP TABLE workflow_template_steps_backup;
DROP TABLE workflow_step_executions_backup;

SELECT '=== 外键约束移除完成 ===' as info;