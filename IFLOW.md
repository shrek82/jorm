# jorm 项目上下文文档

## 项目概述

jorm 是一个轻量级、高性能的 Go 语言 ORM（对象关系映射）库，支持链式操作、事务管理和多种数据库。该项目旨在提供一个简洁、高效的数据库操作接口，支持主流数据库系统（MySQL、PostgreSQL、SQLite、Oracle、SQL Server），并具有良好的可扩展性。

## 项目架构

### 核心组件

1. **core/** - ORM 核心实现
   - `db.go`: 数据库连接池和主入口点
   - `query.go`: 查询构建器和执行器
   - `builder.go`: SQL 语句构建器
   - `tx.go`: 事务管理
   - `hooks.go`: 钩子函数接口定义
   - `errors.go`: 错误定义和处理
   - `migration.go`: 表结构自动迁移功能
   - `preload.go`: 关系预加载功能

2. **dialect/** - 数据库方言实现
   - `dialect.go`: 方言接口定义
   - `mysql.go`: MySQL 方言实现
   - `postgres.go`: PostgreSQL 方言实现
   - `sqlite3.go`: SQLite 方言实现
   - `oracle.go`: Oracle 方言实现
   - `sqlserver.go`: SQL Server 方言实现

3. **model/** - 模型定义和元数据
   - `model.go`: 模型结构定义
   - `field.go`: 字段定义
   - `relation.go`: 关系定义
   - `tag.go`: 标签解析

4. **logger/** - 日志系统
   - `logger.go`: 日志接口和实现

5. **pool/** - 连接池管理
   - `pool.go`: 连接池接口和实现

6. **query/** - 查询子句
   - `clause.go`: 查询子句定义

7. **tests/** - 测试文件
   - 各种集成和单元测试

## 核心功能

### 1. 模型定义
- 支持使用 `jorm` 标签定义模型字段属性
- 自动处理字段名到列名的蛇形命名转换
- 支持主键、自增、唯一约束、默认值等属性
- 支持外键关系定义
- 支持自动时间戳（创建时间、更新时间）

### 2. 查询构建器
- 链式 API 设计，支持流畅的查询构建
- 支持 WHERE、JOIN、ORDER BY、LIMIT、OFFSET 等子句
- 支持 IN 条件和 OR 条件
- 支持聚合函数（COUNT、SUM、AVG等）
- 支持原生SQL查询和执行

### 3. 多数据库支持
- 支持 MySQL、PostgreSQL、SQLite、Oracle、SQL Server 等主流数据库
- 通过方言接口可轻松扩展其他数据库支持
- 自动处理不同数据库的 SQL 语法差异
- 统一的API接口，便于在不同数据库间切换

### 4. 事务管理
- 手动事务管理
- 函数式事务（推荐），自动处理提交和回滚
- 支持带Context的事务操作

### 5. 钩子函数
- 支持 `BeforeInsert`、`AfterInsert`、`BeforeUpdate`、`AfterUpdate`、`AfterFind` 等钩子
- 允许在数据库操作前后执行自定义逻辑

### 6. 连接池
- 内置连接池管理
- 可配置最大连接数、空闲连接数、连接生命周期等参数
- 支持连接超时和生命周期管理

### 7. 自动迁移
- 基于模型定义自动创建表结构
- 支持字段类型、约束、索引的自动创建
- 支持表结构的增量更新

### 8. Context支持
- 支持超时控制和操作取消
- 提供更细粒度的操作控制
- 防止长时间运行的查询阻塞应用

## 构建和运行

### 依赖管理
- 使用 Go Modules 进行依赖管理
- 主要依赖 `github.com/mattn/go-sqlite3` 用于 SQLite 支持
- 支持其他数据库驱动（如MySQL、PostgreSQL等）

### Go版本
- 项目基于 Go 1.25.3 开发

### 构建命令
```bash
# 初始化项目
go mod tidy

# 运行测试
go test ./tests/...

# 运行基准测试
go test -bench=.

# 运行特定测试
go test -v ./tests/integration_test.go
```

### 测试
- 项目包含全面的集成测试
- 包括构建器测试、查询测试、并发测试、方言测试等
- 使用多种数据库进行测试以确保兼容性

## 使用模式

### 模型定义
```go
type User struct {
    ID        int64     `jorm:"pk;auto"`
    Name      string    `jorm:"size:100 notnull"`
    Email     string    `jorm:"size:100 unique"`
    Age       int       `jorm:"default:0"`
    CreatedAt time.Time `jorm:"auto_time"`
    UpdatedAt time.Time `jorm:"auto_update"`
}

type Order struct {
    ID        int64     `jorm:"pk;auto"`
    UserID    int64     `jorm:"fk:User.ID"`
    Amount    float64   `jorm:"notnull"`
    Status    string    `jorm:"size:20 default:'pending'"`
    CreatedAt time.Time `jorm:"auto_time"`
}
```

### 基本 CRUD 操作
```go
// 插入
user := &User{Name: "Alice", Email: "alice@example.com", Age: 25}
id, err := db.Model(user).Insert(user)

// 查询
var u User
err = db.Model(&User{}).Where("id = ?", id).First(&u)

// 更新
affected, err := db.Model(&User{}).
    Where("id = ?", id).
    Update(map[string]any{"age": 26})

// 删除
affected, err = db.Model(&User{}).Where("id = ?", id).Delete()
```

### 事务操作
```go
err := db.Transaction(func(tx *core.Tx) error {
    user := &User{Name: "Alice", Email: "alice@example.com"}
    id, err := tx.Model(user).Insert(user)
    if err != nil {
        return err // 自动回滚
    }
    
    order := &Order{UserID: id, Amount: 100.0, Status: "pending"}
    _, err = tx.Model(order).Insert(order)
    if err != nil {
        return err // 自动回滚
    }
    
    return nil // 自动提交
})
```

## 设计模式

### 1. 构建器模式
- `sqlBuilder` 通过 `sync.Pool` 实现对象池，提高性能

### 2. 方言模式
- 通过 `dialect.Dialect` 接口实现不同数据库的适配

### 3. 钩子模式
- 通过接口实现允许用户自定义操作前后的行为

### 4. 适配器模式
- `Executor` 接口统一 `*sql.DB` 和 `*sql.Tx` 的操作

### 5. 流式接口模式
- 链式调用设计，提供流畅的API体验

## 性能优化

1. **对象池**: 使用 `sync.Pool` 缓存 `sqlBuilder` 对象
2. **扫描计划缓存**: 缓存字段扫描映射以避免重复反射
3. **连接池**: 有效管理数据库连接，减少连接创建开销
4. **批量操作**: 支持批量插入以提高性能
5. **预加载优化**: 支持关系预加载，避免N+1查询问题

## 错误处理

- 定义了结构化的错误类型 `JormError`
- 提供了常见的错误类型如 `ErrRecordNotFound`、`ErrDuplicateKey`、`ErrForeignKey` 等
- 支持错误包装和类型检查
- 提供了详细的错误信息以便调试

## 开发约定

### 代码风格
- 使用 Go 的标准格式化和命名约定
- 广泛使用接口定义，便于扩展和测试
- 通过标签系统配置模型属性

### 测试实践
- 提供全面的集成测试覆盖
- 使用多种数据库进行测试
- 测试各种数据库操作场景
- 包括并发测试和性能测试

### 文档
- 详细的 README.md 包含所有功能说明
- USAGE.md 提供丰富的使用示例
- 代码注释遵循 Go 文档约定
- 提供最佳实践指南

## 扩展性

### 自定义方言
- 通过实现 `dialect.Dialect` 接口可添加新的数据库支持
- 抽象了SQL语法差异，便于扩展

### 自定义日志
- 支持多种日志格式和级别
- 可自定义日志输出格式

### 钩子扩展
- 提供丰富的钩子函数点
- 支持在操作前后执行自定义逻辑