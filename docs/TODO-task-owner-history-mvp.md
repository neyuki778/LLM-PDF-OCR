# Tasks 鉴权 + 历史任务（MVP）TODO

## 1. 目标

- 登录用户可以查看自己的历史任务 ID（长期有效，不依赖前端本地保存）。
- `tasks` 相关接口具备基础 owner 校验，避免跨用户读取任务。
- 保持现有主流程（上传 -> 切分 -> OCR -> 聚合）不大改。

## 2. MVP 范围

- 包含：
  - `POST /api/tasks` 写入 owner 信息
  - `GET /api/tasks/:id` owner 校验
  - `GET /api/tasks/:id/result` owner 校验
  - 新增 `GET /api/tasks/history` 返回当前登录用户历史任务
- 不包含：
  - 复杂 RBAC/多角色权限系统
  - 游客历史任务追踪
  - 任务删除与级联清理
  - 历史列表的复杂筛选/搜索

## 3. 数据设计（Redis）

基于当前 Redis + AOF 持久化，新增/调整以下 key：

1. `task:{task_id}`
- 任务主记录（已有）
- 补充字段：`owner_user_id`、`created_at`（若缺失则补齐）

2. `user_tasks:{user_id}`（新增）
- 类型：`ZSET`
- `score`: 任务创建时间戳（Unix）
- `member`: `task_id`
- 用途：按时间倒序查询某用户的历史任务

## 4. TTL 策略（MVP）

- 历史任务需要长期可见，因此以下数据不应短 TTL 过期：
  - `task:{task_id}`（至少保留较长时间，MVP 可先不设 TTL）
  - `user_tasks:{user_id}`
- 若保留运行态缓存 TTL，需确保“历史元数据”不会随缓存一起丢失。

## 5. 接口行为约定

1. `POST /api/tasks`
- 登录用户：创建任务后写 `owner_user_id=user.id`，并写入 `user_tasks:{user_id}`。
- 游客：`owner_user_id` 为空（兼容现状）。

2. `GET /api/tasks/:id`
- 当任务存在 owner：
  - 未登录：`401`
  - 非 owner：`404`（推荐，避免枚举）或 `403`
  - owner 本人：`200`
- 当任务无 owner（历史/游客）：
  - MVP 先兼容放行

3. `GET /api/tasks/:id/result`
- 校验逻辑与 `GET /api/tasks/:id` 一致。

4. `GET /api/tasks/history?limit=20&cursor=<ts>`
- 仅登录用户可访问。
- 默认按创建时间倒序返回 task IDs（可附带 `status`、`created_at`）。
- MVP 可先做简单分页（`limit` + 时间游标）。

## 6. 实施 TODO

1. 数据模型与存储
- [x] `TaskRecord` 增加 `owner_user_id`
- [x] `SaveTask/GetTask` 序列化兼容新字段
- [x] 新增 user history 索引读写（`ZADD/ZREVRANGEBYSCORE`）

2. 任务创建写入链路
- [x] `createTask` 获取当前 `userID`
- [x] 创建成功后写入 owner 与 user history 索引
- [x] 写入失败时记录日志（不影响主流程或按需失败返回，MVP 先明确策略）

3. 查询鉴权链路
- [x] 在 `getTask/getResult` 中统一做 owner 校验
- [x] 无 access 但有 refresh 的场景维持 `401`（配合前端自动 refresh）
- [x] 非 owner 返回 `404`（或 `403`，二选一）

4. 历史任务 API
- [ ] 新增 `GET /api/tasks/history`
- [ ] 返回字段最小化：`task_id/status/created_at`
- [ ] 接口参数校验：`limit` 上限保护

5. 前端最小接入
- [ ] 登录状态下请求 `/api/tasks/history`
- [ ] 在右上角用户区域或独立区块展示最近任务 ID 列表
- [ ] 点击可回填查询框或直接跳转任务详情

## 7. 风险与取舍

- 旧任务没有 `owner_user_id`：MVP 先按“无 owner 放行”兼容。
- Redis 体量增长：后续可做归档/冷存储（MVP 暂不处理）。
- 多实例一致性：Redis 天然共享，优于仅内存映射方案。

## 8. 验收标准（MVP DoD）

- 登录用户可看到自己的历史任务 ID，服务重启后仍可见。
- 已登录用户不能读取他人 owner 任务。
- 未登录用户无法访问 owner 任务。
- 现有上传与处理主链路不回归。
