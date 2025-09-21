-- 事务保证测试脚本
-- 验证数据一致性

.echo on
.headers on

SELECT '=== 事务保证测试 ===' as info;

-- 检查数据完整性
SELECT '=== 检查数据完整性 ===' as info;

-- 检查是否有孤立的模板步骤（引用不存在的步骤）
SELECT 'workflow_template_steps 引用不存在的步骤:' as check_type, COUNT(*) as count
FROM workflow_template_steps ts 
LEFT JOIN steps s ON ts.step_id = s.id 
WHERE s.id IS NULL;

-- 检查是否有孤立的模板步骤（引用不存在的模板）
SELECT 'workflow_template_steps 引用不存在的模板:' as check_type, COUNT(*) as count
FROM workflow_template_steps ts 
LEFT JOIN workflow_templates t ON ts.template_id = t.id 
WHERE t.id IS NULL;

-- 检查是否有孤立的执行记录（引用不存在的模板）
SELECT 'workflow_executions 引用不存在的模板:' as check_type, COUNT(*) as count
FROM workflow_executions e 
LEFT JOIN workflow_templates t ON e.template_id = t.id 
WHERE t.id IS NULL;

-- 检查是否有孤立的步骤执行记录（引用不存在的执行）
SELECT 'workflow_step_executions 引用不存在的执行:' as check_type, COUNT(*) as count
FROM workflow_step_executions se 
LEFT JOIN workflow_executions e ON se.execution_id = e.id 
WHERE e.id IS NULL;

-- 检查是否有孤立的步骤执行记录（引用不存在的步骤）
SELECT 'workflow_step_executions 引用不存在的步骤:' as check_type, COUNT(*) as count
FROM workflow_step_executions se 
LEFT JOIN steps s ON se.step_id = s.id 
WHERE s.id IS NULL;

-- 统计各表的记录数
SELECT '=== 数据统计 ===' as info;

SELECT '步骤总数:' as table_name, COUNT(*) as count FROM steps;
SELECT '模板总数:' as table_name, COUNT(*) as count FROM workflow_templates;
SELECT '模板步骤总数:' as table_name, COUNT(*) as count FROM workflow_template_steps;
SELECT '执行记录总数:' as table_name, COUNT(*) as count FROM workflow_executions;
SELECT '步骤执行记录总数:' as table_name, COUNT(*) as count FROM workflow_step_executions;

SELECT '=== 测试完成 ===' as info;