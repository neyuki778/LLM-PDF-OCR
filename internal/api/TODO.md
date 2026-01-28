# HTTP API TODO

## Phase 4.1 - 最小可用服务

- [ ] 初始化 Gin server (`server.go`)
- [ ] POST /api/tasks - 上传 PDF，创建任务
- [ ] GET /api/tasks/:id - 查询任务状态

## Phase 4.2 - 完善功能

- [ ] GET /api/tasks/:id/result - 下载 Markdown 结果
- [ ] DELETE /api/tasks/:id - 取消/删除任务
- [ ] 统一错误响应格式
- [ ] 请求参数验证

## Phase 4.3 - 优化

- [ ] 文件大小限制 (MaxMultipartMemory)
- [ ] 上传文件类型校验 (目前只允许 PDF)
- [ ] 临时文件清理机制
- [ ] 请求日志中间件
