# jorm

一个轻量级、高性能的 Go 语言 ORM（对象关系映射）库，支持链式操作、事务管理和多种数据库。

## 特性

- **轻量高效**：核心依赖最少，代码精简，性能优化
- **链式操作**：流畅的 API 设计，支持链式调用
- **类型安全**：基于反射的动态类型处理，编译时检查
- **多数据库支持**：支持 MySQL、PostgreSQL、SQLite、Oracle、SQL Server
- **事务管理**：提供函数式事务支持，自动提交/回滚
- **钩子函数**：支持 Before/After 操作钩子
- **连接池**：内置连接池管理，支持重试机制
- **自动迁移**：支持基于模型的表结构自动创建和更新
- **关联预加载**：支持 BelongsTo、HasOne、HasMany、ManyToMany 关系预加载
- **数据验证**：内置数据验证器，支持多种验证规则
- **Context 支持**：支持超时控制和操作取消

## 安装

```bash
go get github.com/shrek82/jorm
```

## 快速示例

```go
package main

import (
    "github.com/shrek82/jorm/core"
    _ "github.com/mattn/go-sql-driver/mysql"
)

type User struct {
    ID        int64     `jorm:"pk auto"`
    Name      string    `jorm:"size:100 notnull"`
    Email     string    `jorm:"size:100 unique"`
    CreatedAt time.Time `jorm:"auto_time"`
}

func main() {
    db, err := core.Open("mysql", "user:password@/dbname", &core.Options{
        MaxOpenConns: 10,
        MaxIdleConns: 5,
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()

    db.AutoMigrate(&User{})

    user := &User{Name: "Alice", Email: "alice@example.com"}
    id, _ := db.Model(user).Insert(user)
    fmt.Println("Inserted ID:", id)
}
```

## 详细文档

查看 [docs/](./docs/) 目录获取完整的使用手册：

- [快速开始](./docs/01-快速开始.md)
- [数据库连接](./docs/02-数据库连接.md)
- [模型定义](./docs/03-模型定义.md)
- [查询操作](./docs/04-查询操作.md)
- [插入操作](./docs/05-插入操作.md)
- [更新操作](./docs/06-更新操作.md)
- [删除操作](./docs/07-删除操作.md)
- [事务处理](./docs/08-事务处理.md)
- [钩子函数](./docs/09-钩子函数.md)
- [关联关系](./docs/10-关联关系.md)
- [数据验证](./docs/11-数据验证.md)
- [迁移管理](./docs/12-迁移管理.md)
- [日志配置](./docs/13-日志配置.md)
- [Context支持](./docs/14-Context支持.md)
- [原生SQL](./docs/15-原生SQL.md)
- [错误处理](./docs/16-错误处理.md)
- [性能优化](./docs/17-性能优化.md)
- [最佳实践](./docs/18-最佳实践.md)

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
