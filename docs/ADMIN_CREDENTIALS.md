# 🎫 管理员账户信息

**创建日期**: 2026-05-23
**最后更新**: 2026-05-23

---

## 🔐 管理员登录凭据

### 管理后台登录

**访问地址**: http://localhost:5175

**账户信息**:
- **邮箱**: `admin@example.com`
- **密码**: `admin123`
- **角色**: 平台管理员 (role = 2)

---

## 👥 角色权限说明

| 角色 | Role值 | 权限说明 |
|------|--------|----------|
| **普通用户** | 0 | 参与竞拍、查看商品、管理自己的订单 |
| **主播** | 1 | 创建和管理竞拍、管理自己的商品、查看竞拍数据 |
| **平台管理员** | 2 | 管理所有用户、商品、竞拍，查看系统统计 |

---

## 🔑 测试账户

### 普通用户账户
可自行注册或使用测试账户（如果已创建）。

---

## ⚠️ 安全提示

### 生产环境建议

1. **修改默认密码**
   ```sql
   -- 在MySQL中更新密码
   UPDATE users
   SET password = '$2a$10$YOUR_NEW_HASH'
   WHERE id = 999;
   ```

2. **使用强密码**
   - 至少12位
   - 包含大小写字母、数字、特殊字符

3. **定期更换密码**
   - 建议每3个月更换一次

4. **启用双因素认证**（未来功能）

---

## 📝 密码重置

### 通过应用重置（开发中）
目前需要通过数据库直接修改。

### 通过数据库重置
```bash
# 生成新密码的bcrypt哈希
# 例如: "newpassword123"

# 更新数据库
docker exec -i dy-ai-live-auction-mysql mysql -u root -proot live_auction <<EOF
UPDATE users
SET password = '$2a$10$YOUR_NEW_BCRYPT_HASH'
WHERE id = 999;
EOF
```

---

## 🔧 账户管理

### 查看管理员账户
```bash
docker exec -i dy-ai-live-auction-mysql mysql -u root -proot live_auction -e \
  "SELECT id, name, email, role FROM users WHERE role = 2;"
```

### 创建新管理员
```bash
# 通过API创建（推荐）
curl -X POST http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "新管理员",
    "email": "newadmin@example.com",
    "password": "secure_password",
    "role": 2
  }'
```

---

## ✅ 验证登录

### 方式1: Web界面
1. 访问 http://localhost:5175
2. 输入邮箱和密码
3. 应该能成功登录并看到管理面板

### 方式2: API测试
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'

# 应该返回JWT token和用户信息
```

---

## 📚 相关文档

- [角色权限修复文档](./ROLE_FIX_DOCUMENTATION.md)
- [端口配置说明](./PORT_CONFIGURATION.md)
- [系统访问指南](../MVP_ACCESS_GUIDE.md)

---

**重要**: 请妥善保管管理员账户信息，不要在生产环境中使用默认密码！
