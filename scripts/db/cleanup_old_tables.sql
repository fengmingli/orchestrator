-- 数据库表清理脚本
-- 删除已废弃的旧模型对应的表
-- 这些表对应的模型已从代码中移除

-- 检查表是否存在数据（安全检查）
.echo on
.headers on

SELECT '=== 清理前检查旧表数据 ===' as info;

SELECT 'templates count: ' || COUNT(*) as check_result FROM templates;
SELECT 'template_steps count: ' || COUNT(*) as check_result FROM template_steps;  
SELECT 'executions count: ' || COUNT(*) as check_result FROM executions;
SELECT 'step_executions count: ' || COUNT(*) as check_result FROM step_executions;

SELECT '=== 开始删除旧表 ===' as info;

-- 删除旧表（按依赖关系顺序）
DROP TABLE IF EXISTS step_executions;
DROP TABLE IF EXISTS template_steps;
DROP TABLE IF EXISTS executions; 
DROP TABLE IF EXISTS templates;

SELECT '=== 检查剩余表 ===' as info;

-- 显示剩余的表
.tables

SELECT '=== 清理完成 ===' as info;