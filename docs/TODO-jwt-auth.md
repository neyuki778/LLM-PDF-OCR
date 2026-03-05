# JWT 登录功能改造 TODO（SQLite 版）

## 0. 目标与边界

- 目标：为现有 OCR 服务增加「邮箱 + 密码」注册登录，并用 JWT 保护业务接口。
- 约束：
  - 先做最小可用版本（MVP），不做邮箱验证码、忘记密码、第三方 OAuth。
  - 数据库存储使用 SQLite。
  - 前端沿用现有静态页面，先支持基础登录态。

## 1. 鉴权方案（建议）

- Token 设计：
  - Access Token：有效期短（建议 15 分钟），用于访问受保护 API。
  - Refresh Token：有效期长（建议 7 天），用于换新 Access Token。
- 存储方式：
  - 优先使用 HttpOnly Cookie（同域部署下最省心，降低 XSS 读取风险）。
  - Access Token 与 Refresh Token 都走 Cookie（或 Access 用 Header、Refresh 用 Cookie，二选一）。
- 算法：
  - JWT 签名先用 HS256（单服务足够）。
  - 生产环境要求高强度 `JWT_SECRET`（32 字节以上随机串）。

## 2. 数据库设计（SQLite）

新增两张表：

1. `users`
- `id` TEXT PRIMARY KEY（UUID）
- `email` TEXT NOT NULL UNIQUE
- `password_hash` TEXT NOT NULL
- `created_at` DATETIME NOT NULL
- `updated_at` DATETIME NOT NULL

2. `refresh_tokens`
- `id` TEXT PRIMARY KEY（UUID）
- `user_id` TEXT NOT NULL
- `token_hash` TEXT NOT NULL UNIQUE
- `expires_at` DATETIME NOT NULL
- `revoked_at` DATETIME NULL
- `created_at` DATETIME NOT NULL
- 外键：`user_id -> users.id`
- 索引：`idx_refresh_tokens_user_id`、`idx_refresh_tokens_expires_at`

说明：
- 不直接存明文 refresh token，只存哈希值（泄露后风险更低）。
- 登录时写入 refresh token 记录；登出时标记 revoked。

## 3. 后端模块拆分（建议）

建议在 `internal/` 下新增：

- `internal/auth/`
  - `types.go`：claims、请求/响应结构
  - `service.go`：注册/登录/刷新/登出业务
  - `jwt.go`：签发与解析 token
  - `password.go`：bcrypt 哈希与校验
  - `middleware.go`：Gin 鉴权中间件
- `internal/store/sqlite/`（或扩展现有 keystore 模块）
  - `db.go`：连接初始化、建表
  - `user_repo.go`：users CRUD
  - `refresh_repo.go`：refresh token CRUD

## 4. API 设计（MVP）

新增接口：

1. `POST /api/auth/register`
- 入参：`email`, `password`
- 行为：创建用户，密码哈希后入库
- 返回：用户基础信息（不回密码、不回 hash）

2. `POST /api/auth/login`
- 入参：`email`, `password`
- 行为：校验密码，签发 access/refresh token，写 cookie
- 返回：登录成功信息（可带 user profile）

3. `POST /api/auth/refresh`
- 入参：无（从 cookie 取 refresh token）
- 行为：校验 refresh token + DB 记录，签发新 access token（可选 token rotation）
- 返回：刷新成功

4. `POST /api/auth/logout`
- 行为：撤销当前 refresh token（DB 标记 revoked），清除 cookie
- 返回：登出成功

5. `GET /api/auth/me`
- 行为：验证 access token，返回当前用户信息

## 5. 现有 OCR API 的保护范围

建议分阶段：

1. 第一阶段（低风险）
- 保持现有 OCR 接口不变：
  - `POST /api/tasks`
  - `GET /api/tasks/:id`
  - `GET /api/tasks/:id/result`
- 先把 auth 全链路跑通。

2. 第二阶段（上线前）
- 给任务接口加鉴权中间件。
- 任务数据增加 `owner_user_id` 字段，查询时做 owner 校验，避免越权读取。

## 6. 配置项（.env）

新增：

- `JWT_SECRET`
- `JWT_ISSUER`（默认 `llm-pdf-ocr`）
- `JWT_ACCESS_TTL`（默认 `15m`）
- `JWT_REFRESH_TTL`（默认 `168h`）
- `AUTH_COOKIE_SECURE`（本地可 `false`，生产 `true`）
- `SQLITE_PATH`（默认 `./data/app.db`）

## 7. 安全基线

- 密码：`bcrypt`（cost 使用默认或 12）。
- 注册/登录都做参数校验：
  - email 格式合法
  - password 最小长度（建议 >= 8）
- 统一错误文案：登录失败不区分“邮箱不存在/密码错误”。
- JWT 必须校验：签名、过期时间、issuer、token type。
- Refresh token rotation（建议）：每次 refresh 使旧 token 失效并下发新 token。

## 8. 实施 TODO（按顺序）

1. 建表与 SQLite 初始化
- [x] 新增 SQLite 连接与 migrations
- [x] 创建 `users`、`refresh_tokens` 表和索引

2. 用户仓储层
- [x] 按 email 查用户
- [x] 创建用户（邮箱唯一约束处理）
- [x] refresh token 的新增/查询/撤销

3. Auth Service
- [ ] 注册：hash 密码 + 入库
- [ ] 登录：校验密码 + 生成 token + 保存 refresh token
- [ ] 刷新：校验 refresh token + rotation
- [ ] 登出：撤销 refresh token + 清 cookie

4. Gin 中间件与路由
- [ ] 新增 `/api/auth/*` 路由
- [ ] 实现 `AuthMiddleware`
- [ ] 注入到受保护路由分组

5. 前端最小改造
- [ ] 新增登录/注册入口（可先做简易表单）
- [ ] 登录后调用 `/api/auth/me` 显示当前用户
- [ ] 未登录时提示并禁用受保护操作

6. 测试
- [ ] 单元测试：密码、JWT、service 逻辑
- [ ] 集成测试：注册→登录→调用受保护接口→刷新→登出

## 9. 验收标准（Definition of Done）

- 可以完成完整流程：
  - 注册成功
  - 登录成功
  - 使用 access token 调受保护接口成功
  - access 过期后可 refresh
  - logout 后 refresh 失效
- 关键安全要求达成：
  - 数据库中无明文密码
  - 数据库中无明文 refresh token
  - 无 token 时访问受保护接口返回 401

## 10. 已知取舍（MVP）

- 暂不做邮箱验证与密码找回。
- 暂不做多因子认证。
- 暂不做登录频率限制与验证码（可作为下一阶段安全增强）。
