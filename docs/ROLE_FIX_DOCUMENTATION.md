# 角色定义修复文档

**修复日期**: 2026-05-23
**修复原因**: 角色定义不一致导致权限检查错误

---

## 📋 问题描述

### 角色定义不一致

**后端Model定义** (`backend/auction/model/user.go`):
```go
const (
    RoleUser     Role = 0 // 普通用户
    RoleStreamer Role = 1 // 主播
    RoleAdmin    Role = 2 // 平台管理员
)
```

**实际使用问题**:
1. 数据库迁移脚本创建管理员时使用 `role = 1`
2. 前端检查管理员时判断 `role === 1`
3. 后端 `IsAdmin()` 方法检查 `role == 2`

**结果**: 管理员账户无法通过后端的 `IsAdmin()` 检查！

---

## 🔧 修复内容

### 1. 前端修复

**文件**: `frontend/admin/src/store/authContext.tsx`

**修改**:
- Line 34: `parsedUser.role === 1` → `parsedUser.role === 2`
- Line 47: `newUser.role !== 1` → `newUser.role !== 2`
- Line 71: `isAdmin: user?.role === 1` → `isAdmin: user?.role === 2`

### 2. 数据库迁移脚本修复

**文件**: `scripts/migrations/002_add_auth_fields.sql`

**修改**:
- Line 10: 注释更新为 `角色: 0=普通用户, 1=主播, 2=平台管理员`
- 演示管理员不再由迁移脚本创建，统一由 `scripts/init-demo-users.sh` seed，避免出现第二账号来源。

### 3. 后端Model修复

**文件**: `backend/product/model/user.go`

**修改**:
- 添加 `Role` 常量定义（与auction service保持一致）
- 更新 `IsAdmin()` 方法: `u.Role == 1` → `u.Role == int(RoleAdmin)`
- 添加 `IsStreamer()` 和 `CanManageProduct()` 方法

---

## ✅ 角色定义统一

修复后的角色定义：

| Role值 | 角色 | 说明 |
|--------|------|------|
| 0 | RoleUser | 普通用户 - 可参与竞拍 |
| 1 | RoleStreamer | 主播 - 可创建和管理竞拍 |
| 2 | RoleAdmin | 平台管理员 - 拥有所有权限 |

---

## 🎫 管理员账户

**邮箱**: `admin@example.com`
**手机号**: `13800138003`
**密码**: `Demo@123456`
**角色**: `2` (平台管理员)

---

## 📝 执行数据库修复

### 步骤1: 连接数据库
```bash
mysql -h 127.0.0.1 -P 3306 -u root -proot auction
```

### 步骤2: 执行修复脚本
```bash
source /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/migrations/fix_admin_role.sql
```

或直接执行SQL:
```sql
UPDATE users SET role = 2 WHERE id = 9104 AND email = 'admin@example.com';
SELECT id, name, email, role FROM users WHERE id = 9104;
```

---

## ⚠️ 注意事项

### 权限检查逻辑

**正确的权限层级**:
```
RoleAdmin (2) > RoleStreamer (1) > RoleUser (0)
```

**中间件检查** (`backend/auction/middleware/rbac.go`):
```go
// 管理员可以操作所有资源
if userRole >= 2 {
    c.Next(ctx)
    return
}

// 主播只能操作自己的资源
if userRole == 1 {
    // 检查资源所有权
}
```

### 权限范围

**普通用户 (role=0)**:
- 参与竞拍
- 查看商品
- 管理自己的订单

**主播 (role=1)**:
- 创建和管理竞拍
- 管理自己的商品
- 查看竞拍数据

**平台管理员 (role=2)**:
- 管理所有用户
- 管理所有商品
- 管理所有竞拍
- 查看系统统计

---

## ✅ 修复验证

### 1. 验证数据库
```sql
SELECT id, name, email, role FROM users WHERE id = 9104;
-- 应该显示 role = 2
```

### 2. 验证前端登录
```bash
# 访问管理后台
open http://localhost:5175

# 使用管理员账户登录
# 邮箱: admin@example.com
# 手机号: 13800138003
# 密码: Demo@123456

# 应该能成功登录并看到管理界面
```

### 3. 验证后端权限
```bash
# 获取JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800138003","password":"Demo@123456"}'

# 使用token访问管理API
curl -X GET http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <token>"

# 应该能成功访问
```

---

## 🔄 回滚方案

如果修复出现问题，可以回滚：

```sql
UPDATE users SET role = 1 WHERE id = 9104 AND email = 'admin@example.com';
```

然后恢复前端和后端代码的修改。

---

**修复人**: Claude Code
**修复状态**: ✅ 已完成
**下一步**: 执行数据库修复脚本，测试管理员登录
