# Feature Specification: 商品管理与竞拍系统优化 - 直播间关联

**Feature**: `20260523-product-auction-live`
**Created**: 2026-05-23
**Status**: Draft
**Input**: User confirmed brainstorm requirements for product management optimization, auction management enhancement, and live stream integration

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 商品发布到直播间 (Priority: P1)

商家（主播）登录管理后台，在商品管理页面找到草稿状态的商品，点击"发布"按钮。系统检查商家是否有对应的直播间（每个商家有且仅有一个直播间），如果直播间状态正常，则将商品状态从"草稿"改为"已发布"，并自动创建一个"待开始"状态的竞拍记录。该竞拍记录关联到商家的直播间，用户端可以在直播间的竞拍商品列表中看到这个商品。

**Why this priority**: 这是核心业务流程，直接影响商家能否正常开展拍卖业务。没有这个功能，商家无法将商品上架到直播间进行拍卖。

**Technical Implementation**:

**前端实现**：
- 在商品管理列表页（`frontend/admin/src/pages/Product/List.tsx`）的操作列中，为草稿状态商品添加"发布"按钮
- 点击发布按钮触发API调用：`POST /api/v1/products/{id}/publish`
- 发布成功后刷新商品列表，商品状态显示为"已发布"
- 优化配置规则表单UI（`frontend/admin/src/pages/Product/RuleConfig.tsx`），使用统一的UI组件库，增强用户体验

**后端实现**：
- 新增API端点：`POST /api/v1/products/{id}/publish`（Product Service）
- 实现逻辑：
  1. 验证商品状态为"草稿"
  2. 查询商家的直播间（通过creator_id关联）
  3. 创建竞拍记录（Auction表），状态设为"待开始"（status=0）
  4. 更新商品状态为"已发布"
  5. 返回成功响应

**数据模型**：
- Product表：新增`status`字段值 2（已下架），原状态：0=草稿，1=已发布
- Auction表：已有`creator_id`字段关联商家，新增`live_stream_id`字段关联直播间
- LiveStream表（新增）：
  ```sql
  CREATE TABLE live_streams (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    creator_id BIGINT NOT NULL UNIQUE COMMENT '商家ID，一对一',
    name VARCHAR(128) NOT NULL COMMENT '直播间名称',
    description TEXT COMMENT '直播间描述',
    cover_image VARCHAR(256) COMMENT '封面图',
    status TINYINT DEFAULT 1 COMMENT '状态：0=禁用，1=正常',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_creator_id (creator_id)
  );
  ```

**Independent Test**: 可以通过创建一个测试商品，点击发布按钮，验证商品状态变更和竞拍记录创建，检查数据库中live_streams和auctions表的关联关系。

**Acceptance Scenarios**:

1. **Given** 商品状态为"草稿"，**When** 商家点击"发布"按钮，**Then** 商品状态变为"已发布"，创建待开始的竞拍记录，关联到商家直播间
2. **Given** 商品状态为"已发布"，**When** 商家点击"发布"按钮，**Then** 系统提示"商品已发布，无法重复发布"
3. **Given** 商家没有直播间，**When** 商家点击"发布"按钮，**Then** 系统自动创建直播间后发布商品

---

### User Story 2 - 商品下架功能 (Priority: P1)

商家在商品管理页面找到已发布的商品，点击"下架"按钮。系统检查该商品是否有进行中的竞拍，如果有，提示确认是否中断拍卖流程。确认后，将商品状态改为"已下架"，同时取消所有关联的待开始和进行中的竞拍记录，并向已出价的用户发送通知。

**Why this priority**: 商品下架是必要的业务操作，直接影响用户体验和资金安全。需要妥善处理进行中的竞拍，避免纠纷。

**Technical Implementation**:

**前端实现**：
- 在商品管理列表页，为已发布状态商品添加"下架"按钮
- 点击下架按钮时，检查是否有进行中的竞拍，如有则弹出确认对话框
- 确认后调用API：`POST /api/v1/products/{id}/unpublish`
- 下架成功后刷新商品列表，商品状态显示为"已下架"

**后端实现**：
- 新增API端点：`POST /api/v1/products/{id}/unpublish`（Product Service）
- 实现逻辑：
  1. 验证商品状态为"已发布"
  2. 查询关联的竞拍记录（通过product_id）
  3. 如果有进行中或待开始的竞拍，发送通知给已出价的用户
  4. 取消关联的竞拍记录（status=4 已取消）
  5. 更新商品状态为"已下架"（status=2）
  6. 返回成功响应

**通知机制**：
- 使用现有的Notification系统
- 通知内容："您参与竞拍的商品[商品名]已被商家下架"

**Independent Test**: 可以通过创建一个已发布的商品，创建竞拍记录，添加一些出价记录，然后点击下架按钮，验证竞拍取消和通知发送。

**Acceptance Scenarios**:

1. **Given** 商品状态为"已发布"，无进行中竞拍，**When** 商家点击"下架"按钮，**Then** 商品状态变为"已下架"
2. **Given** 商品状态为"已发布"，有待开始竞拍，**When** 商家确认下架，**Then** 商品下架，竞拍取消，已出价用户收到通知
3. **Given** 商品状态为"已发布"，有进行中竞拍，**When** 商家确认下架，**Then** 商品下架，竞拍中断，已出价用户收到通知

---

### User Story 2.5 - 用户关注直播间功能 (Priority: P1)

用户可以在用户端浏览直播间列表，选择关注感兴趣的直播间。关注后，当该直播间发布新商品或竞拍即将开始时，用户会收到推送通知。同时，用户参与过竞拍（已出价）的商品也会收到相关通知（下架、竞拍结束等）。两种通知机制独立运作，用户可以管理自己的通知偏好。

<!-- clarify: 2026-05-23 — Added user follow live stream feature based on Q&A -->

**Why this priority**: 用户关注直播间是提升用户粘性和活跃度的关键功能，直接影响商品曝光率和竞拍参与度。混合通知模式确保用户不会错过重要信息。

**Technical Implementation**:

**前端实现**：
- **MVP阶段**：
  - 用户端新增独立的直播间列表页面，展示所有直播间及其关注状态
  - 用户端首页展示推荐直播间（根据关注数量、竞拍热度排序）
  - 在直播间详情页和直播间竞拍列表页添加"关注/取消关注"按钮
- **完整阶段（后续迭代）**：
  - 在竞拍商品详情页显示所属直播间信息，支持快速关注
  - 个性化推荐：根据用户历史关注、出价行为推荐相关直播间
- 用户中心新增"我的关注"页面，管理关注的直播间
- 通知设置页面：允许用户配置通知偏好（新商品发布、竞拍开始、竞拍结束等）

<!-- clarify: 2026-05-23 — Split frontend implementation into MVP and full phases -->

**后端实现**：
- 新增API端点（Auction Service）：
  - `POST /api/v1/live-streams/{id}/follow`: 关注直播间
  - `DELETE /api/v1/live-streams/{id}/follow`: 取消关注
  - `GET /api/v1/user/followed-live-streams`: 获取用户关注的直播间列表
  - `GET /api/v1/live-streams/{id}/followers/stats`: 获取直播间关注统计数据（商家可访问）
- 关注统计数据（仅统计，不暴露具体用户信息）：
  - 关注用户总数
  - 新增关注数（今日、本周、本月）
  - 关注用户活跃度分布（近7天活跃、近30天活跃、不活跃）
  - 关注用户参与竞拍统计（参与过竞拍的关注用户数）
- 通知触发时机：
  - 商品发布时：通知关注该直播间的用户
  - 竞拍即将开始（提前30分钟）：通知关注用户（分批次推送，最长耗时10分钟，确保在竞拍开始前所有用户收到通知）
  - 竞拍结束：通知已出价用户
  - 商品下架：通知已出价用户

<!-- clarify: 2026-05-23 — Changed notification timing from 5min to 30min before auction start, added batch push constraint -->

**数据模型**：
- 新增 `user_live_stream_follows` 中间表：
  ```sql
  CREATE TABLE user_live_stream_follows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    live_stream_id BIGINT NOT NULL COMMENT '直播间ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '关注时间',
    notification_enabled TINYINT DEFAULT 1 COMMENT '是否接收通知：0=否，1=是',
    UNIQUE KEY uk_user_live_stream (user_id, live_stream_id),
    INDEX idx_live_stream_id (live_stream_id),
    INDEX idx_user_id (user_id)
  );
  ```

**通知性能优化**：
- **MVP阶段**：
  - 使用消息队列（Redis/RabbitMQ）异步发送通知
  - 全量推送策略：所有通知都发送，不做去重
  - 通知列表按时间倒序排列，最新的在前
- **二期迭代**：
  - 用户可选择通知模式：全量接收、智能聚合、仅重要通知
  - 智能聚合：短时间内来自同一直播间的多条通知聚合为一条
  - 优先级过滤：竞拍开始>竞拍结束>新商品发布>其他
- 分批次推送策略：
  - 单个直播间关注用户超过1万时启用分批推送
  - 每批推送1万用户，批次间隔3-5秒
  - 最大耗时控制在10分钟内（100万用户需10批，约50秒）
- 时序约束：
  - 竞拍开始前30分钟触发通知，确保推送完成后再开始竞拍
  - 如果推送队列积压，优先处理竞拍开始通知（优先级最高）
- 限流策略：对单个直播间的通知频率进行限制（每天最多10条推送通知）
- 多渠道策略：站内信实时显示，推送通知异步发送

<!-- clarify: 2026-05-23 — Split notification strategy into MVP and Phase 2, added user-configurable mode for Phase 2 -->

**Independent Test**: 可以通过用户账户关注一个直播间，商家发布新商品，验证用户是否收到通知。取消关注后再次发布商品，验证不再收到通知。

**Acceptance Scenarios**:

1. **Given** 用户浏览直播间列表，**When** 点击"关注"按钮，**Then** 关注成功，按钮变为"已关注"
2. **Given** 用户已关注直播间A，**When** 商家发布新商品，**Then** 用户收到新商品发布通知
3. **Given** 用户已关注直播间A，**When** 直播间竞拍即将开始（提前30分钟），**Then** 用户收到竞拍开始提醒
4. **Given** 用户参与过竞拍，**When** 商品下架或竞拍结束，**Then** 用户收到相关通知（无论是否关注该直播间）
5. **Given** 用户在通知设置中关闭某直播间的通知，**When** 该直播间发布新商品，**Then** 用户不收到通知，但关注关系保留
6. **Given** 直播间有100万关注用户，**When** 商家发布新商品，**Then** 系统分10批推送通知，总耗时不超过10分钟

<!-- clarify: 2026-05-23 — Updated scenario 3 timing and added scenario 6 for performance test -->

---

### User Story 2.6 - 用户出价竞拍 (Priority: P0)

用户在直播间竞拍页面看到正在进行的竞拍商品，点击"出价"按钮，输入出价金额。系统验证用户是否已登录（通过JWT token），如果未登录则提示登录。已登录用户的出价金额必须大于当前最高价+最小加价幅度，系统记录出价信息，更新竞拍排名，并通过WebSocket实时推送给出价用户和其他参与者。

**Why this priority**: 这是竞拍系统的核心功能，直接影响用户能否参与竞拍。没有用户出价功能，竞拍业务无法闭环。用户认证是出价功能的前置条件，必须严格验证。

<!-- clarify: 2026-05-23 — Added user bid story with authentication requirement based on Q&A -->

**Technical Implementation**:

**前端实现**：
- 在竞拍详情页（`frontend/h5/src/pages/Live/index.tsx`）添加出价按钮和出价输入框
- 点击出价按钮时检查用户登录状态：
  - 未登录：跳转到登录页面或弹出登录框
  - 已登录：显示出价输入框
- 出价金额验证：
  - 必须 > 当前最高价 + 最小加价幅度
  - 实时提示最小出价金额
- 调用API：`POST /api/v1/auctions/{id}/bids`
- WebSocket订阅竞拍更新，实时显示最新排名

**后端实现**：
- API端点：`POST /api/v1/auctions/{id}/bids`（Auction Service）
- 认证要求：**必须携带有效的JWT token**
- 实现逻辑：
  1. **从JWT上下文获取用户ID**（`c.Get("user_id")`）
  2. 如果JWT中无user_id，返回401错误："未认证，请先登录"
  3. 验证竞拍状态为"进行中"或"延时中"
  4. 验证出价金额 > 当前最高价 + 最小加价幅度
  5. 验证竞拍未结束（end_time > now）
  6. 创建出价记录（Bid表）
  7. 更新竞拍当前价格
  8. 如果触发延时条件，延长竞拍时间
  9. 通过WebSocket推送更新给出价用户和其他参与者
  10. 返回出价成功响应

**认证流程**：
```go
// 从JWT上下文获取用户ID
var userID int64
userIDInterface, exists := c.Get("user_id")
if !exists {
    c.JSON(401, map[string]interface{}{
        "code":    401,
        "message": "未认证，请先登录",
    })
    return
}
userID = userIDInterface.(int64)
```

**数据模型**：
- Bid表（已有）：
  ```sql
  CREATE TABLE bids (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    auction_id BIGINT NOT NULL COMMENT '竞拍ID',
    user_id BIGINT NOT NULL COMMENT '出价用户ID',
    amount DECIMAL(10,2) NOT NULL COMMENT '出价金额',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_auction_id (auction_id),
    INDEX idx_user_id (user_id)
  );
  ```

**错误处理**：
- 401 未认证：用户未登录，提示"请先登录"
- 400 参数错误：出价金额不符合规则
- 409 竞拍已结束：竞拍已结束，无法出价
- 500 服务器错误：系统内部错误

**Independent Test**: 可以通过创建一个竞拍记录，使用未登录用户尝试出价，验证返回401错误。然后使用已登录用户出价，验证出价成功且记录正确。

**Acceptance Scenarios**:

1. **Given** 用户未登录，**When** 点击"出价"按钮，**Then** 提示"请先登录"，无法出价
2. **Given** 用户已登录，竞拍状态为"进行中"，**When** 出价金额 > 当前最高价+最小加价，**Then** 出价成功，更新排名
3. **Given** 用户已登录，竞拍状态为"已结束"，**When** 尝试出价，**Then** 提示"竞拍已结束，无法出价"
4. **Given** 用户已登录，出价金额 < 当前最高价+最小加价，**When** 尝试出价，**Then** 提示"出价金额不足"
5. **Given** 用户A已出价，**When** 用户B出价更高，**Then** 用户A收到通知"您的出价已被超越"

---

### User Story 3 - 配置规则表单UI优化 (Priority: P2)

商家点击"配置规则"按钮进入规则配置页面，看到优化后的表单界面。表单采用统一的UI组件库，包含清晰的分组、提示文字、输入验证和实时预览功能。表单字段包括：起拍价、加价幅度、封顶价、竞拍时长、延时设置等。填写完成后点击保存，系统验证数据合法性并创建竞拍规则。

**Why this priority**: 当前配置规则表单UI简陋（使用内联样式），用户体验差，容易导致配置错误。优化UI可以提升商家操作效率和减少错误。

**Technical Implementation**:

**前端UI优化**：
- 移除内联样式，使用项目的统一CSS类（参考index.css）
- 表单分组：
  - 基础设置：起拍价、加价幅度、封顶价
  - 时间设置：竞拍时长、延时触发时间
  - 高级设置：单次延时时长、最大延时时长
- 添加输入验证：
  - 加价幅度：必须 > 0
  - 竞拍时长：60-3600秒
  - 封顶价：0表示无封顶
- 添加实时预览：显示竞拍规则摘要
- 使用现代化的卡片式布局，符合整体设计风格

**代码修改**：
- 文件：`frontend/admin/src/pages/Product/RuleConfig.tsx`
- 移除formItemStyle、labelStyle、inputStyle、buttonStyle等内联样式
- 使用项目统一的class命名
- 添加表单验证逻辑
- 优化错误提示方式（使用Toast组件）

**Independent Test**: 可以通过打开配置规则页面，验证表单样式是否符合设计规范，输入非法数据时是否有正确的验证提示。

**Acceptance Scenarios**:

1. **Given** 用户进入配置规则页面，**When** 页面加载完成，**Then** 显示优化后的表单UI，使用统一的组件库
2. **Given** 用户填写加价幅度为0，**When** 点击保存，**Then** 显示错误提示"加价幅度必须大于0"
3. **Given** 用户填写竞拍时长为30秒，**When** 点击保存，**Then** 显示错误提示"竞拍时长至少60秒"

---

### User Story 4 - 竞拍管理状态筛选优化 (Priority: P1)

商家和管理员在竞拍管理页面，看到"全部/待开始/进行中/已结束"四个筛选按钮。点击不同按钮可以筛选对应状态的竞拍记录。管理员视角下，还能看到额外的两列数据：直播间ID和直播间名称，并且可以通过搜索框根据直播间ID或名称搜索竞拍记录。

**Why this priority**: 状态筛选是基础的列表管理功能，直接影响用户查找和管理竞拍的效率。管理员全局视角是权限隔离的重要体现。

**Technical Implementation**:

**前端实现**：
- 修改文件：`frontend/admin/src/pages/Auction/List.tsx`
- 新增筛选按钮：
  - 当前：`['all', 'ongoing', 'ended']`
  - 改为：`['all', 'pending', 'ongoing', 'ended']`
- 根据用户角色显示不同列：
  - 商家：显示现有列（竞拍ID、商品名称、当前价、出价次数、状态、剩余时间、中标者、操作）
  - 管理员：额外显示直播间ID、直播间名称两列
- 新增搜索框（管理员视角）：
  - 支持按直播间ID或名称搜索
  - 实时过滤列表数据

**后端实现**：
- 修改API：`GET /api/v1/auctions`（Auction Service）
- 新增查询参数：
  - `status`: 0=待开始, 1=进行中, 2=延时中, 3=已结束
  - `live_stream_id`: 按直播间ID筛选
  - `live_stream_name`: 按直播间名称模糊搜索
- 返回数据新增字段（管理员视角）：
  - `live_stream_id`: 直播间ID
  - `live_stream_name`: 直播间名称
- 权限控制：
  - 商家：只返回自己直播间的竞拍记录（通过creator_id过滤）
  - 管理员：返回所有竞拍记录

**数据查询优化**：
- Auction表需要JOIN LiveStream表获取直播间名称
- 为live_stream_id添加索引

**Independent Test**: 可以通过管理员账户登录，验证筛选按钮是否正确，是否能看到直播间列，搜索功能是否正常。再通过商家账户登录，验证只能看到自己的竞拍记录。

**Acceptance Scenarios**:

1. **Given** 商家登录竞拍管理页面，**When** 点击"待开始"按钮，**Then** 显示status=0的竞拍记录，且都是该商家直播间的
2. **Given** 管理员登录竞拍管理页面，**When** 页面加载完成，**Then** 显示所有竞拍记录，包含直播间ID和名称列
3. **Given** 管理员在搜索框输入直播间名称，**When** 输入完成，**Then** 列表实时过滤显示匹配的竞拍记录

---

### User Story 5 - 直播间管理模块 (Priority: P2)

管理员可以在管理后台看到"直播间管理"菜单，点击进入直播间列表页面。页面显示所有直播间的列表，包含：直播间ID、直播间名称、商家信息、当前竞拍数、历史成交额、状态等。管理员可以对直播间进行启用/禁用操作，查看直播间详情和统计数据。

**Why this priority**: 直播间是新增的核心实体，需要独立的管理模块。管理员需要能够查看和管理所有直播间，维护平台秩序。

**Technical Implementation**:

**前端实现**：
- 新增页面：`frontend/admin/src/pages/LiveStream/List.tsx`
- 路由配置：`/live-streams`
- 导航菜单：添加"直播间管理"菜单项
- 列表展示：
  - 直播间ID、名称、商家名称
  - 当前竞拍数（进行中+待开始）
  - 历史成交额
  - 状态（正常/禁用）
  - 操作（查看详情、启用/禁用）
- 详情页面：`frontend/admin/src/pages/LiveStream/Detail.tsx`
  - 直播间基本信息
  - 竞拍历史列表
  - 统计数据图表

**后端实现**：
- 新增API端点（Product Service）：
  - `GET /api/v1/live-streams`: 获取直播间列表
  - `GET /api/v1/live-streams/{id}`: 获取直播间详情
  - `PUT /api/v1/live-streams/{id}/status`: 更新直播间状态
- 权限控制：仅管理员可访问

**数据模型**：
- LiveStream表（已在User Story 1中定义）
- 统计查询：
  - 当前竞拍数：COUNT(auctions WHERE live_stream_id=? AND status IN (0,1,2))
  - 历史成交额：SUM(auctions.current_price WHERE live_stream_id=? AND status=3)

**Independent Test**: 可以通过管理员账户登录，进入直播间管理页面，验证列表展示、状态操作和详情查看功能。

**Acceptance Scenarios**:

1. **Given** 管理员进入直播间管理页面，**When** 页面加载完成，**Then** 显示所有直播间列表，包含统计数据
2. **Given** 管理员点击禁用按钮，**When** 确认操作，**Then** 直播间状态变为禁用，该直播间下的所有待开始竞拍被取消
3. **Given** 管理员点击查看详情，**When** 进入详情页，**Then** 显示直播间基本信息、竞拍历史和统计数据

---

### User Story 6 - 权限和数据可见性隔离 (Priority: P1)

系统实现角色权限隔离，确保商家只能看到和管理自己的直播间和竞拍数据，管理员可以看到和管理所有数据。所有涉及跨用户数据访问的API都需要进行权限验证，防止越权访问。

**Why this priority**: 权限隔离是系统安全的基础，防止数据泄露和越权操作，符合业务合规要求。

**Technical Implementation**:

**前端实现**：
- 根据用户角色动态显示/隐藏菜单项和功能按钮
- 在API请求拦截器中自动携带用户身份信息
- 显示当前用户角色和直播间信息

**后端实现**：
- 所有API端点添加权限中间件
- 商家角色（Role=1）：
  - 自动过滤数据：只返回creator_id匹配的数据
  - 验证操作权限：只能操作自己的数据
- 管理员角色（Role=2）：
  - 可访问所有数据
  - 可执行所有操作
- 使用现有的JWT中间件获取用户ID和角色

**权限验证示例**：
```go
// Auction Service
func (h *AuctionHandler) List(ctx context.Context, c *app.RequestContext) {
    userID := c.GetInt64("user_id")
    userRole := c.GetInt("user_role")
    
    if userRole == 1 { // 商家
        // 只查询该商家的直播间下的竞拍
        auctions, err := h.auctionDAO.GetByCreatorID(ctx, userID)
    } else if userRole == 2 { // 管理员
        // 查询所有竞拍
        auctions, err := h.auctionDAO.List(ctx, filters)
    }
}
```

**Independent Test**: 可以通过商家账户和管理员账户分别登录，验证看到的数据范围是否符合权限设计，尝试越权操作是否被拦截。

**Acceptance Scenarios**:

1. **Given** 商家登录系统，**When** 访问竞拍管理页面，**Then** 只看到自己直播间的竞拍记录
2. **Given** 商家尝试访问其他商家的竞拍详情，**When** 直接修改URL中的ID，**Then** 返回403权限不足错误
3. **Given** 管理员登录系统，**When** 访问任何管理页面，**Then** 可以看到所有数据并执行所有操作

---

### Edge Cases

- **直播间自动创建**：商家首次发布商品时，如果还没有直播间，系统自动创建
- **并发发布**：多个商家同时发布商品到同一直播间（理论上不应发生，因为直播间与商家一对一）
- **下架时的通知**：如果有大量用户出价，下架时需要批量发送通知，考虑性能和延迟
- **管理员禁用直播间**：如果直播间有进行中的竞拍，禁用时需要提示并处理这些竞拍
- **商品重复发布**：已下架的商品是否可以再次发布？可以，但会创建新的竞拍记录
- **搜索性能**：管理员按直播间名称搜索时，需要考虑模糊查询的性能优化
- **统计数据一致性**：直播间的竞拍数和成交额统计需要保证实时性和一致性
- **权限继承**：未来如果有子账号功能，需要考虑权限继承和细化

## Requirements *(mandatory)*

### Functional Requirements

**商品管理**：
- **FR-001**: 系统必须允许商家为草稿状态的商品添加"发布"按钮，点击后将商品发布到直播间
- **FR-002**: 系统必须为商品状态新增"已下架"状态（status=2），完善状态流转
- **FR-003**: 系统必须在商家下架商品时，检查并取消关联的待开始和进行中竞拍
- **FR-004**: 系统必须优化配置规则表单UI，使用统一组件库，添加输入验证
- **FR-005**: 系统必须在商品下架时，向已出价的用户发送通知

**竞拍管理**：
- **FR-006**: 系统必须在竞拍管理页面新增"待开始"筛选按钮
- **FR-007**: 系统必须为管理员显示直播间ID和直播间名称两列数据
- **FR-008**: 系统必须允许管理员通过搜索框按直播间ID或名称搜索竞拍记录
- **FR-009**: 系统必须根据用户角色过滤数据，商家只能看到自己直播间的竞拍

**用户出价**：
- **FR-023**: 系统必须要求用户登录后才能参与竞拍出价，未登录用户无法出价
- **FR-024**: 系统必须在用户出价时验证出价金额大于当前最高价+最小加价幅度
- **FR-025**: 系统必须从JWT token中获取用户ID，确保出价记录与真实用户关联
- **FR-026**: 系统必须在出价成功后实时推送更新给所有参与者（通过WebSocket）
- **FR-027**: 系统必须在用户出价被超越时发送通知给出价用户

<!-- clarify: 2026-05-23 — Added FR-023 to FR-027 for user bid authentication requirements -->

**直播间管理**：
- **FR-010**: 系统必须创建LiveStream表，实现直播间与商家一对一关联
- **FR-011**: 系统必须为管理员提供直播间管理模块，包含列表、详情、状态管理功能
- **FR-012**: 系统必须在商家首次发布商品时自动创建直播间（如果不存在）
- **FR-013**: 系统必须统计直播间的当前竞拍数和历史成交额
- **FR-017**: 系统必须创建user_live_stream_follows表，支持用户关注/取消关注直播间
- **FR-018**: 系统必须在商品发布时，通知关注该直播间的用户
- **FR-019**: 系统必须在竞拍即将开始时（提前30分钟），通知关注直播间的用户（采用分批次推送策略，最长耗时10分钟）
<!-- clarify: 2026-05-23 — Updated FR-019 timing from 5min to 30min -->
- **FR-020**: 系统必须支持用户管理通知偏好（开启/关闭特定直播间的通知）
- **FR-021**: 系统必须为商家提供直播间关注统计数据（关注总数、新增趋势、活跃度分布），不暴露具体用户信息以保护隐私
- **FR-022（二期）**: 系统必须支持用户选择通知模式（全量接收、智能聚合、仅重要通知），并按用户偏好处理通知

<!-- clarify: 2026-05-23 — Added FR-022 for Phase 2 user-configurable notification mode -->
<!-- clarify: 2026-05-23 — Added FR-017 to FR-020 for follow feature -->

**权限控制**：
- **FR-014**: 系统必须实现角色权限隔离，商家只能访问自己的数据
- **FR-015**: 系统必须在所有API端点添加权限验证中间件
- **FR-016**: 系统必须防止商家越权访问其他商家的数据

### Key Entities

- **Product（商品）**: 核心实体，新增"已下架"状态，支持发布和下架操作
- **Auction（竞拍）**: 关联商品和直播间，新增live_stream_id字段，状态包括待开始/进行中/延时中/已结束/已取消
- **Bid（出价记录）**: 记录用户在竞拍中的出价信息，包含用户ID、出价金额、时间戳，必须与认证用户关联
- **LiveStream（直播间）**: 新增实体，与商家一对一关联，包含名称、描述、封面图、状态等属性
- **UserLiveStreamFollow（用户关注直播间）**: 新增中间表，记录用户关注直播间的关系，支持通知偏好设置
- **User（用户）**: 角色定义，普通用户（Role=0），主播=商家（Role=1），管理员（Role=2）
- **Notification（通知）**: 用于商品下架、新商品发布、竞拍开始、出价被超越等场景的通知推送

<!-- clarify: 2026-05-23 — Added Bid entity and updated User role to include Role=0 for regular users -->

## Success Criteria *(mandatory)*

### Measurable Outcomes

**用户体验指标**：
- **SC-001**: 商家可以在3步内完成商品发布操作（点击发布→确认→完成）
- **SC-002**: 商家可以在2步内完成商品下架操作（点击下架→确认）
- **SC-003**: 配置规则表单填写时间减少30%（从平均5分钟降至3.5分钟）
- **SC-004**: 管理员搜索直播间竞拍记录的时间不超过2秒

**功能完整性**：
- **SC-005**: 商品状态流转完整覆盖：草稿→已发布→已下架→已发布（可循环）
- **SC-006**: 所有API端点均实现权限验证，无越权访问漏洞
- **SC-007**: 商品下架时，100%的已出价用户都能收到通知
- **SC-008**: 管理员可以查看和管理100%的直播间数据

**数据准确性**：
- **SC-009**: 直播间统计数据（竞拍数、成交额）与实际数据100%一致
- **SC-010**: 商品发布后，用户端实时可见（延迟不超过3秒）

**系统性能**：
- **SC-011**: 商品发布API响应时间不超过500ms
- **SC-012**: 竞拍列表查询（含搜索）响应时间不超过1秒
- **SC-013**: 支持100个商家同时发布商品，无性能瓶颈

**业务指标**：
- **SC-014**: 商家发布商品的成功率达到95%以上（减少操作错误）
- **SC-015**: 商品下架导致的用户投诉率降至最低（通过及时通知）

## Assumptions & Dependencies

### Assumptions
- 直播间与商家是一对一关系，每个商家有且仅有一个直播间
- 主播就是商家，使用同一个角色（Role=1）
- 商品发布后默认创建"待开始"状态的竞拍记录
- 已下架的商品可以再次发布，但会创建新的竞拍记录
- 管理员可以查看所有数据，但不能代替商家操作

### Dependencies
- 需要现有的用户认证系统（JWT）支持
- 需要现有的通知系统支持消息推送
- 需要前端UI组件库支持
- 需要数据库支持新的LiveStream表和相关索引

## Technical Notes

### Architecture Impact
- 前端：需要新增直播间管理模块，修改商品管理和竞拍管理页面
- 后端：Product Service需要新增发布/下架API，Auction Service需要修改查询逻辑
- 数据库：新增LiveStream表，Auction表新增live_stream_id字段

### Security Considerations
- 所有API必须验证用户权限，防止越权访问
- 商品下架操作需要记录操作日志
- 直播间禁用操作需要验证影响范围

### Performance Considerations
- 管理员搜索功能需要优化查询性能，考虑添加索引
- 统计数据查询需要优化，避免实时计算，考虑缓存策略
- 批量通知发送需要考虑消息队列，避免阻塞主流程

### Migration Strategy
- 数据库迁移：创建LiveStream表，为现有商家自动创建直播间
- 代码部署：前后端同步部署，确保API兼容性
- 功能开关：可考虑使用功能开关逐步上线，降低风险

## Out of Scope

以下功能不在本次需求范围内：
- 直播间的高级设置（背景音乐、弹幕管理等）
- 商品的批量操作（批量发布、批量下架）
- 竞拍的快速复制功能
- 商品的审核流程
- 子账号和权限细化管理
- 直播间的粉丝管理功能
