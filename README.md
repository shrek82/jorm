# jorm

一个轻量级、高性能的 Go 语言 ORM（对象关系映射）库，支持链式操作、事务管理和多种数据库。

## 特性

- **轻量高效**：核心依赖最少，代码精简，性能优化
- **链式操作**：流畅的 API 设计，支持链式调用
- **类型安全**：基于反射的动态类型处理，编译时检查
- **多数据库支持**：支持 MySQL、PostgreSQL、SQLite 等
- **事务管理**：提供手动事务和函数式事务两种方式
- **钩子函数**：支持 Before/After 操作钩子
- **连接池**：内置连接池管理，支持自定义配置
- **自动迁移**：支持基于模型的表结构自动创建
- **Context 支持**：支持超时控制和操作取消

## 安装

```bash
go get github.com/shrek82/jorm
```

## 快速开始

### 1. 定义模型

```go
type User struct {
    ID        int64     `jorm:"pk auto"`
    Name      string    `jorm:"size:100 notnull"`
    Email     string    `jorm:"size:100 unique"`
    Age       int       `jorm:"default:0"`
    CreatedAt time.Time `jorm:"auto_time"`
    UpdatedAt time.Time `jorm:"auto_update"`
}
```

### 2. 初始化数据库

```go
import (
    "github.com/shrek82/jorm/core"
    _ "github.com/go-sql-driver/mysql"
)

db, err := core.Open("mysql", "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4", &core.Options{
    MaxOpenConns:    10,
    MaxIdleConns:    5,
    ConnMaxLifetime: time.Hour,
})
if err != nil {
    panic(err)
}
defer db.Close()

// 自动创建表
err = db.AutoMigrate(&User{})
if err != nil {
    panic(err)
}
```

### 3. CRUD 操作

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

## 详细用法

### 模型标签

`jorm` 标签用于定义字段的数据库行为：

| 标签 | 说明 | 示例 |
|------|------|------|
| `pk` | 主键 | `jorm:"pk"` |
| `auto` | 自增 | `jorm:"pk auto"` |
| `column` | 列名 | `jorm:"column:email_addr"` |
| `size` | 字段大小 | `jorm:"size:100"` |
| `unique` | 唯一索引 | `jorm:"unique"` |
| `notnull` | 非空 | `jorm:"notnull"` |
| `default` | 默认值 | `jorm:"default:'pending'"` |
| `fk` | 外键 | `jorm:"fk:User.ID"` |
| `auto_time` | 插入时自动设置时间 | `jorm:"auto_time"` |
| `auto_update` | 更新时自动设置时间 | `jorm:"auto_update"` |

### 数据库连接

```go
// MySQL
db, err := core.Open("mysql", "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4", opts)

// PostgreSQL
db, err := core.Open("postgres", "host=localhost port=5432 user=postgres dbname=app sslmode=disable", opts)

// SQLite
db, err := core.Open("sqlite3", "./app.db", opts)
```

连接池配置：

```go
&core.Options{
    MaxOpenConns:    100,                  // 最大打开连接数
    MaxIdleConns:    10,                   // 最大空闲连接数
    ConnMaxLifetime: time.Hour,            // 连接最大生命周期
    ConnMaxIdleTime: time.Minute * 10,     // 空闲连接最大存活时间
}
```

### 查询操作

#### 基础查询

```go
// 查询单条记录
var user User
err := db.Model(&User{}).Where("id = ?", 1).First(&user)

// 查询多条记录
var users []User
err = db.Model(&User{}).Where("age > ?", 20).Find(&users)

// 查询所有记录
err = db.Model(&User{}).Find(&users)

// 统计数量
count, err := db.Model(&User{}).Where("age > ?", 20).Count()

// 求和
sum, err := db.Model(&User{}).Sum("age")
```

#### 条件查询

```go
// WHERE 条件
db.Model(&User{}).
    Where("age > ?", 20).
    Where("name LIKE ?", "%Alice%").
    Find(&users)

// OR 条件
db.Model(&User{}).
    Where("age > ?", 20).
    OrWhere("name = ?", "Alice").
    Find(&users)

// IN 条件
db.Model(&User{}).
    WhereIn("id", []int64{1, 2, 3}).
    Find(&users)
```

#### 排序和分页

```go
// 排序
db.Model(&User{}).
    OrderBy("age DESC").
    OrderBy("name ASC").
    Find(&users)

// 分页
page := 1
pageSize := 10
offset := (page - 1) * pageSize

db.Model(&User{}).
    OrderBy("created_at DESC").
    Limit(pageSize).
    Offset(offset).
    Find(&users)
```

#### 字段选择

```go
// 选择指定字段
db.Model(&User{}).
    Select("id", "name", "email").
    Find(&users)

// 聚合查询
type Result struct {
    Count   int64   `jorm:"column:count"`
    AvgAge  float64 `jorm:"column:avg_age"`
}
var result Result
db.Model(&User{}).
    Select("COUNT(*) as count", "AVG(age) as avg_age").
    Scan(&result)
```

### 插入操作

```go
// 插入单条记录
user := &User{
    Name:  "Alice",
    Email: "alice@example.com",
    Age:   25,
}
id, err := db.Model(user).Insert(user)

// 批量插入
users := []*User{
    {Name: "Bob", Email: "bob@example.com", Age: 30},
    {Name: "Charlie", Email: "charlie@example.com", Age: 28},
}
count, err := db.Model(&User{}).BatchInsert(users)
```

### 更新操作

```go
// 更新指定字段
affected, err := db.Model(&User{}).
    Where("id = ?", 1).
    Update(map[string]any{
        "name": "Alice Updated",
        "age":  26,
    })

// 更新整个模型
user.Name = "Alice Smith"
affected, err = db.Model(&User{}).
    Where("id = ?", user.ID).
    Update(user)
```

### 删除操作

```go
// 删除单条记录
affected, err := db.Model(&User{}).
    Where("id = ?", 1).
    Delete()

// 批量删除
affected, err = db.Model(&User{}).
    Where("age < ?", 18).
    Delete()
```

### 原生 SQL

```go
// 执行原生查询
var users []User
err := db.Raw("SELECT * FROM user WHERE age > ?", 20).Scan(&users)

// 执行原生命令
result, err := db.Exec("UPDATE user SET age = age + 1 WHERE id = ?", 1)
rowsAffected, _ := result.RowsAffected()
```

## 事务

### 手动事务管理

```go
tx, err := db.Begin()
if err != nil {
    panic(err)
}

// 在事务中执行操作
user := &User{Name: "Alice", Email: "alice@example.com"}
id, err := tx.Model(user).Insert(user)
if err != nil {
    tx.Rollback()
    panic(err)
}

order := &Order{UserID: id, Amount: 100.0, Status: "pending"}
_, err = tx.Model(order).Insert(order)
if err != nil {
    tx.Rollback()
    panic(err)
}

// 提交事务
if err := tx.Commit(); err != nil {
    panic(err)
}
```

### 函数式事务（推荐）

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
if err != nil {
    panic(err)
}
```

## 钩子函数

通过实现钩子接口，可以在数据库操作前后执行自定义逻辑：

```go
type User struct {
    ID        int64     `jorm:"pk auto"`
    Name      string    `jorm:"size:100"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// BeforeInsert 插入前钩子
func (u *User) BeforeInsert() error {
    if u.Name == "" {
        return errors.New("name is required")
    }
    u.CreatedAt = time.Now()
    return nil
}

// AfterInsert 插入后钩子
func (u *User) AfterInsert(id int64) error {
    u.ID = id
    fmt.Printf("User %s inserted with ID: %d\n", u.Name, id)
    return nil
}

// BeforeUpdate 更新前钩子
func (u *User) BeforeUpdate() error {
    u.UpdatedAt = time.Now()
    return nil
}

// AfterUpdate 更新后钩子
func (u *User) AfterUpdate() error {
    fmt.Printf("User %s updated\n", u.Name)
    return nil
}

// AfterFind 查询后钩子
func (u *User) AfterFind() error {
    // 可以在这里进行数据转换等操作
    return nil
}
```

## Context 支持

```go
import "context"

// 带超时的查询
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var users []User
err := db.Model(&User{}).
    WithContext(ctx).
    Where("age > ?", 20).
    Find(&users)
if err != nil {
    if err == context.DeadlineExceeded {
        fmt.Println("查询超时")
    }
}

// 可取消的操作
ctx, cancel = context.WithCancel(context.Background())
go func() {
    time.Sleep(2 * time.Second)
    cancel() // 取消操作
}()

err = db.Model(&User{}).
    WithContext(ctx).
    Find(&users)
if err == context.Canceled {
    fmt.Println("操作被取消")
}
```

## 日志配置

```go
import "github.com/shrek82/jorm/logger"

// 使用标准日志
db.SetLogger(logger.NewStdLogger())

// 自定义日志
db.SetLogger(&logger.StdLoggerConfig{
    Level:      logger.LogLevelInfo,
    Format:     "json", // 或 "text"
    TimeFormat: "2006-01-02 15:04:05",
})
```

## 错误处理

```go
import "errors"

var user User
err := db.Model(&User{}).Where("id = ?", 1).First(&user)
if err != nil {
    if errors.Is(err, core.ErrRecordNotFound) {
        fmt.Println("记录不存在")
    } else if errors.Is(err, core.ErrDuplicateKey) {
        fmt.Println("重复的键")
    } else {
        fmt.Printf("查询错误: %v\n", err)
    }
}
```

## 注意事项

### 1. 命名约定

- 表名：结构体名自动转换为蛇形命名（如 `User` → `user`，`UserProfile` → `user_profile`）
- 列名：字段名自动转换为蛇形命名（如 `UserID` → `user_id`，`CreatedAt` → `created_at`）

### 2. 性能优化

- **批量操作优先**：使用 `BatchInsert` 而非循环插入
- **限制查询字段**：使用 `Select` 只查询需要的字段
- **使用索引查询**：避免全表扫描
- **连接池配置**：根据负载调整 `MaxOpenConns` 和 `MaxIdleConns`
- **Context 超时**：为长时间查询设置合理的超时时间

### 3. 并发安全

- `DB` 对象是并发安全的，可以在多个 goroutine 中共享
- `Query` 对象不是并发安全的，每个查询应使用新的实例
- `Tx` 对象不是并发安全的，不应在多个 goroutine 中共享

### 4. 事务处理

- 函数式事务是推荐的方式，能自动处理提交和回滚
- 手动事务必须确保在出错时调用 `Rollback()`
- 避免在事务中执行长时间运行的操作

### 5. 错误处理

- 始终检查返回的错误
- 使用 `errors.Is()` 检查特定错误类型
- 钩子函数中的错误会中止操作

### 6. 模型定义

- 结构体字段必须是可导出的（首字母大写）
- 主键字段建议使用 `int64` 类型
- 使用指针类型处理可能为 NULL 的字段

### 7. 数据库兼容性

- 不同数据库的类型映射可能不同
- 外键约束的支持程度因数据库而异
- 批量操作的语法因数据库而异

## 最佳实践

### 1. 重用 Query 对象

```go
// 创建基础查询器
userQuery := db.Model(&User{})

var user1, user2 User
userQuery.Where("id = ?", 1).First(&user1)
userQuery.Where("id = ?", 2).First(&user2)
```

### 2. 批量操作优先

```go
// 推荐：批量插入
db.Model(&User{}).BatchInsert(users)

// 不推荐：循环插入
for _, user := range users {
    db.Model(&User{}).Insert(user)
}
```

### 3. 限制查询字段

```go
// 推荐：只查询需要的字段
db.Model(&User{}).Select("id", "name").Find(&users)

// 不推荐：查询所有字段
db.Model(&User{}).Find(&users)
```

### 4. 使用 Context 超时控制

```go
// 推荐：为长时间查询设置超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

db.Model(&User{}).WithContext(ctx).Find(&users)
```

## 示例项目

完整示例请参考 [USAGE.md](./USAGE.md)

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
