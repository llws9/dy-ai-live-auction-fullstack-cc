-- 修复管理员角色问题
-- Created: 2026-05-23
-- Purpose: 修复管理员账户的role值，从1改为2（平台管理员）

-- 更新现有管理员账户的role
UPDATE users
SET role = 2
WHERE id = 999 AND email = 'admin@example.com';

-- 确保管理员账户正确
SELECT id, name, email, role,
       CASE
           WHEN role = 0 THEN '普通用户'
           WHEN role = 1 THEN '主播'
           WHEN role = 2 THEN '平台管理员'
           ELSE '未知'
       END as role_name
FROM users
WHERE id = 999;
