# Spec: 修复主机添加时的唯一索引冲突 (Partial Index)

## 1. 背景与问题描述
在 OpsPilot 系统中，`nodes` 表定义了一个复合唯一索引 `idx_provider_instance`，涵盖了 `provider` 和 `provider_instance_id` 两列。
目前的实现中，手动添加的主机（非云厂商导入）这两列通常为空字符串或 `NULL`。
由于 PostgreSQL 的复合唯一索引对于 `(NULL, NULL)` 的多条记录处理方式（默认允许），以及可能存在的旧版本 `NOT NULL` 约束或空字符串 `""` 存储，导致在添加第二个手动主机时触发 `duplicate key value violates unique constraint "idx_provider_instance"` 错误。

## 2. 目标
- 允许手动添加多个主机（不强制要求 `provider` 和 `provider_instance_id` 唯一）。
- 保留云厂商导入主机的唯一性校验（防止同一个云实例被重复导入）。
- 确保方案在 PostgreSQL 环境下优雅实现。

## 3. 设计方案：部分索引 (Partial Index)

### 3.1 模型定义修改
修改 `internal/modules/host/model/node.go` 中的 `Node` 结构体，利用 GORM 的 `where` 标签定义部分索引。

```go
type Node struct {
    // ... 其他字段
    Provider   *string `gorm:"column:provider;type:varchar(32);uniqueIndex:idx_provider_instance,where:provider IS NOT NULL AND provider != ''" json:"provider"`
    ProviderID *string `gorm:"column:provider_instance_id;type:varchar(128);uniqueIndex:idx_provider_instance,where:provider IS NOT NULL AND provider != ''" json:"provider_instance_id"`
    // ... 其他字段
}
```

### 3.2 数据库迁移策略
由于 `auto_migrate` 可能无法自动处理索引的 `WHERE` 子句变更（取决于 GORM 版本和驱动实现），我们需要执行一次手动迁移脚本或在代码中显式处理。

**迁移逻辑：**
1. **清理存量数据**：将 `nodes` 表中 `provider` 为空字符串 `""` 的记录更新为 `NULL`。
2. **删除旧索引**：`DROP INDEX IF EXISTS idx_provider_instance;`
3. **创建部分索引**：
   ```sql
   CREATE UNIQUE INDEX idx_provider_instance 
   ON nodes(provider, provider_instance_id) 
   WHERE provider IS NOT NULL AND provider != '';
   ```

## 4. 影响评估
- **向下兼容性**：现有云主机记录不受影响，其唯一性校验依然生效。
- **性能**：部分索引比全量索引更小，查询性能略有提升（虽然规模较小时不明显）。
- **风险**：如果用户切换回不支持部分索引的数据库（如极旧版本的 SQLite），可能需要重新调整方案。但当前环境明确为 PostgreSQL。

## 5. 验证计划

### 5.1 自动化测试
1. **场景一：手动主机并存**
   - 创建主机 A (provider=nil, provider_id=nil)。
   - 创建主机 B (provider=nil, provider_id=nil)。
   - 预期结果：均创建成功。
2. **场景二：云主机重复校验**
   - 创建主机 C (provider="aliyun", provider_id="ins-123")。
   - 尝试创建主机 D (provider="aliyun", provider_id="ins-123")。
   - 预期结果：创建 D 失败，报唯一键冲突。

### 5.2 手动验证
- 通过 UI 连续添加两台 IP 不同的手动主机，观察是否报错。

## 6. 后续行动
1. 提交本设计文档。
2. 编写实施计划（Task List）。
3. 按照计划执行代码修改和数据迁移。
