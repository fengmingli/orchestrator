-- 数据库清理结果验证脚本
.echo on
.headers on

SELECT '=== 数据库清理结果验证 ===' as info;

-- 检查剩余的表
SELECT '=== 当前数据库表 ===' as info;
.tables

-- 验证新表是否存在
SELECT '=== 验证新表结构 ===' as info;

-- 检查新表是否存在
SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✅ steps表存在'
        ELSE '❌ steps表不存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='steps';

SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✅ workflow_templates表存在'
        ELSE '❌ workflow_templates表不存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='workflow_templates';

SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✅ workflow_executions表存在'
        ELSE '❌ workflow_executions表不存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='workflow_executions';

SELECT 
    CASE 
        WHEN COUNT(*) > 0 THEN '✅ distributed_locks表存在'
        ELSE '❌ distributed_locks表不存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='distributed_locks';

-- 验证旧表是否已删除
SELECT '=== 验证旧表已删除 ===' as info;

SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ templates表已删除'
        ELSE '❌ templates表仍存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='templates';

SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ template_steps表已删除'
        ELSE '❌ template_steps表仍存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='template_steps';

SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ executions表已删除'
        ELSE '❌ executions表仍存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='executions';

SELECT 
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ step_executions表已删除'
        ELSE '❌ step_executions表仍存在'
    END as check_result
FROM sqlite_master 
WHERE type='table' AND name='step_executions';

SELECT '=== 验证完成 ===' as info;