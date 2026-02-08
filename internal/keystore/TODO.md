# Keystore TODO

目标：为 Gemini 多 API Key 轮询提供 SQLite 存储 + 内存轮询 KeyProvider。

## 存储层
- [ ] 使用 SQLite（modernc.org/sqlite）
- [ ] 表结构：id/key/note/enabled/created_at/updated_at
- [ ] CRUD：add/list/delete
- [ ] 列表返回 masked key（例如 AIza...9f）
- [ ] 为空时明确返回 no keys

## 轮询层
- [ ] KeyProvider 接口（NextKey/Count）
- [ ] 内存环（round-robin）
- [ ] 支持 refresh（新增/删除后刷新）

## 验证
- [ ] 使用 key 调用 Gemini Models.Get 进行合法性校验
