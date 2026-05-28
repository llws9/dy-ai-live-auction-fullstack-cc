# Research: 后端API补充与测试数据生成

**Feature**: `20260528-backend-api-seed`
**Date**: 2026-05-28

## 研究任务清单

根据 Technical Context 中的未知点和技术依赖，完成以下研究。

---

## 1. Gateway路由注册模式

**Task**: 确认Gateway路由注册方式和权限中间件使用

### Decision
使用现有 `handler.NewProxyHandler(cfg.Services.ProductURL)` 创建代理，通过 `middleware.RequireAdmin()` / `middleware.RequireMerchant()` 控制权限。

### Rationale
- Gateway已有成熟的代理转发模式
- 权限中间件已在 `middleware/permission.go` 实现
- 新路由只需复制现有模式

### Alternatives Considered
- 直接在Product Service处理权限 → 拒绝：Gateway统一入口更符合架构设计

### Evidence
```go
// 现有模式参考 (router/router.go)
authGroup.POST("/products/:id/publish", middleware.RequireMerchant(), productProxy.Forward)
authGroup.GET("/statistics/overview", middleware.RequireAdmin(), productProxy.Forward)
```

---

## 2. Product Service Handler实现模式

**Task**: 确认Handler/Service/DAO分层结构和路由注册方式

### Decision
遵循现有分层结构：Handler → Service → DAO → Model，路由在 `main.go` 的 `registerRoutes` 函数注册。

### Rationale
- Product Service已有多套Handler实现可参考
- 分层结构清晰，便于测试和维护

### Alternatives Considered
- 简化为Handler直接操作DAO → 拒绝：不符合现有架构规范

### Evidence
```go
// 现有模式参考 (product/main.go)
v1.GET("/statistics/overview", statisticsHandler.GetOverview)
```

---

## 3. GORM模型定义规范

**Task**: 确认GORM模型定义、AutoMigrate使用和索引创建

### Decision
- 模型使用 `gorm` tag 定义字段属性
- 唯一索引使用 `uniqueIndex`
- AutoMigrate在 `main.go` 启动时自动执行

### Rationale
- 项目已使用GORM，无需引入新ORM
- AutoMigrate支持增量迁移，不破坏现有数据

### Alternatives Considered
- 手动SQL迁移 → 拒绝：GORM AutoMigrate更便捷且项目已有实践

### Evidence
```go
// 现有模式参考 (model/user.go)
type User struct {
    ID    int64  `json:"id" gorm:"primaryKey;autoIncrement"`
    Email *string `json:"email,omitempty" gorm:"type:varchar(128);uniqueIndex"`
}
```

---

## 4. 逻辑外键实现方式

**Task**: 确认无物理外键的数据关联和一致性保证方式

### Decision
- 使用 `gorm:"index"` 创建索引加速查询
- 应用层通过Service校验关联一致性
- 删除时检查关联数据，禁止删除有依赖的记录

### Rationale
- 高并发场景物理外键会增加写入开销
- 应用层校验更灵活，可配合缓存优化

### Alternatives Considered
- 物理外键 → 拒绝：用户明确要求避免高并发性能问题

---

## 5. Seed脚本数据库连接方式

**Task**: 确认Seed脚本如何连接数据库和处理事务

### Decision
- 使用项目现有的 `dao.InitDBFromEnv()` 初始化连接
- 事务包裹整个生成过程，失败时回滚
- 数据库连接配置从环境变量读取

### Rationale
- 复用现有DAO初始化逻辑
- 环境变量配置已在 `.env` 定义

### Alternatives Considered
- 独立配置文件 → 拒绝：复用现有配置更简洁

---

## 6. 测试数据多样性实现

**Task**: 确认如何生成多样性的测试数据

### Decision
- 使用固定种子确保可复现
- 预定义数据池（名称、邮箱模板、商品示例）
- 按配置比例分布状态和角色

### Rationale
- 固定种子便于调试和验证
- 预定义数据池确保真实性

---

## 研究结论

所有技术点已明确，无需额外澄清。可进入Phase 1设计阶段。