# jorm 用户使用示例

## 快速开始

### 1. 定义模型

```go
package models

import "time"

type User struct {
    ID        int64     `jorm:"pk auto"`
    Name      string    `jorm:"size:100 notnull"`
    Email     string    `jorm:"size:100 unique"`
    Age       int       `jorm:"default:0"`
    CreatedAt time.Time `jorm:"auto_time"`
    UpdatedAt time.Time `jorm:"auto_update"`
}

type Order struct {
    ID        int64     `jorm:"pk auto"`
    UserID    int64     `jorm:"fk:User.ID"`
    Amount    float64   `jorm:"notnull"`
    Status    string    `jorm:"size:20 default:'pending'"`
    CreatedAt time.Time `jorm:"auto_time"`
}
```

### 2. 初始化数据库连接

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/shrek82/jorm/core"
    "github.com/shrek82/jorm/dialect"
    _ "github.com/go-sql-driver/mysql"
    jorm "github.com/shrek82/jorm/core"
)

func main() {
    db, err := jorm.Open("mysql", "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4", &jorm.Options{
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: time.Hour,
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()

    db.SetLogger(jorm.StdLogger)

    // 使用数据库...
}
```

## 基本CRUD操作

### 创建表

```go
// 自动创建User表
err := db.AutoMigrate(&User{})
if err != nil {
    panic(err)
}
```

### 插入数据

```go
// 插入单条记录
user := &User{
    Name:  "Alice",
    Email: "alice@example.com",
    Age:   25,
}
id, err := db.Model(user).Insert(user)
if err != nil {
    panic(err)
}
fmt.Printf("Insert ID: %d\n", id)

// 批量插入
users := []*User{
    {Name: "Bob", Email: "bob@example.com", Age: 30},
    {Name: "Charlie", Email: "charlie@example.com", Age: 28},
}
count, err := db.Model(&User{}).BatchInsert(users)
if err != nil {
    panic(err)
}
fmt.Printf("Batch insert count: %d\n", count)
```

### 查询数据

```go
// 查询单条记录
var user User
err := db.Model(&User{}).Where("id = ?", 1).First(&user)
if err != nil {
    panic(err)
}
fmt.Printf("User: %+v\n", user)

// 查询多条记录
var users []User
err = db.Model(&User{}).Where("age > ?", 20).Find(&users)
if err != nil {
    panic(err)
}
for _, u := range users {
    fmt.Printf("%s: %s\n", u.Name, u.Email)
}

// 查询所有
var allUsers []User
err = db.Model(&User{}).Find(&allUsers)
if err != nil {
    panic(err)
}

// 统计数量
count, err := db.Model(&User{}).Where("age > ?", 20).Count()
if err != nil {
    panic(err)
}
fmt.Printf("Count: %d\n", count)
```

### 更新数据

```go
// 更新指定字段
affected, err := db.Model(&User{}).
    Where("id = ?", 1).
    Update(map[string]any{
        "name": "Alice Updated",
        "age":  26,
    })
if err != nil {
    panic(err)
}
fmt.Printf("Updated %d rows\n", affected)

// 更新整个模型
user.Name = "Alice Smith"
affected, err = db.Model(&User{}).Where("id = ?", user.ID).Update(user)
if err != nil {
    panic(err)
}
```

### 删除数据

```go
// 删除单条记录
affected, err := db.Model(&User{}).Where("id = ?", 1).Delete()
if err != nil {
    panic(err)
}
fmt.Printf("Deleted %d rows\n", affected)

// 批量删除
affected, err = db.Model(&User{}).Where("age < ?", 18).Delete()
if err != nil {
    panic(err)
}
```

## 链式操作

### 条件查询

```go
// WHERE条件
var users []User
err := db.Model(&User{}).
    Where("age > ?", 20).
    Where("name LIKE ?", "%Alice%").
    Find(&users)

// OR条件
err = db.Model(&User{}).
    Where("age > ?", 20).
    OrWhere("name = ?", "Alice").
    Find(&users)

// IN条件
err = db.Model(&User{}).
    WhereIn("id", []int64{1, 2, 3}).
    Find(&users)

// 组合条件
err = db.Model(&User{}).
    Where("age > ?", 20).
    WhereIn("id", []int64{1, 2, 3}).
    OrWhere("name = ?", "Bob").
    Find(&users)
```

### 排序和分页

```go
// 排序
var users []User
err := db.Model(&User{}).
    OrderBy("age DESC").
    OrderBy("name ASC").
    Find(&users)

// 分页
page := 1
pageSize := 10
offset := (page - 1) * pageSize
err = db.Model(&User{}).
    OrderBy("created_at DESC").
    Limit(pageSize).
    Offset(offset).
    Find(&users)

// 统计总数（用于分页）
total, _ := db.Model(&User{}).Count()
pages := (total + pageSize - 1) / pageSize
```

### JOIN操作

```go
// 简单JOIN查询
type OrderWithUser struct {
    Order
    UserName string `jorm:"column:user_name"`
}

var orders []OrderWithUser
err := db.Model(&Order{}).
    Select("order.*", "user.name as user_name").
    Join("user", "user.id = order.user_id").
    Where("order.status = ?", "completed").
    Find(&orders)
```

### 选择字段

```go
// 只查询指定字段
var users []User
err := db.Model(&User{}).
    Select("id", "name", "email").
    Find(&users)

// 聚合查询
var result struct {
    TotalUsers int64   `jorm:"column:count"`
    AvgAge     float64 `jorm:"column:avg_age"`
}
err := db.Model(&User{}).
    Select("COUNT(*) as count", "AVG(age) as avg_age").
    Scan(&result)
```

## 事务操作

### 手动事务管理

```go
// 开启事务
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
// 自动处理提交和回滚
err := db.Transaction(func(tx *jorm.Tx) error {
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

### 带Context的事务

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

tx, err := db.BeginTx(ctx, nil)
if err != nil {
    panic(err)
}

user := &User{Name: "Alice", Email: "alice@example.com"}
    _, err = tx.Model(user).Insert(user)
if err != nil {
    tx.Rollback()
    panic(err)
}

tx.Commit()
```

## 钩子函数

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

## 高级用法

### Context支持

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
ctx, cancel := context.WithCancel(context.Background())
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

// 直接使用Context方法
err = db.Model(&User{}).
    Where("age > ?", 20).
    FindContext(ctx, &users)
```

### 原生SQL

```go
// 执行原生查询
var users []User
err := db.Raw("SELECT * FROM user WHERE age > ?", 20).Scan(&users)

// 执行原生命令
result, err := db.Exec("UPDATE user SET age = age + 1 WHERE id = ?", 1)
rowsAffected, _ := result.RowsAffected()
```

### 子查询

```go
// 使用原生SQL子查询
var users []User
err := db.Model(&User{}).
    Where("id IN (SELECT user_id FROM order WHERE status = ?)", "completed").
    Find(&users)
```

### 批量操作优化

```go
// 分批处理大数据
db.Model(&User{}).
    Where("age > ?", 20).
    Batch(100, func(batch []User) error {
        for _, user := range batch {
            // 处理每批数据
            fmt.Printf("Processing: %s\n", user.Name)
        }
        return nil
    })
```

### 连接池配置

```go
db, err := jorm.Open("mysql", dsn, &jorm.Options{
    MaxOpenConns:    100,  // 最大打开连接数
    MaxIdleConns:    10,   // 最大空闲连接数
    ConnMaxLifetime: time.Hour, // 连接最大生命周期
    ConnMaxIdleTime: time.Minute * 10, // 空闲连接最大存活时间
})
```

## 日志配置

```go
// 使用标准日志
db.SetLogger(jorm.StdLogger)

// 自定义日志格式
db.SetLogger(&jorm.StdLoggerConfig{
    Level:      jorm.LogLevelInfo,
    Format:     "json", // 或 "text"
    TimeFormat: "2006-01-02 15:04:05",
})

// 关闭SQL日志
db.SetLogLevel(jorm.LogLevelSilent)
```

## 错误处理

```go
import "errors"

var user User
err := db.Model(&User{}).Where("id = ?", 1).First(&user)
if err != nil {
    if errors.Is(err, jorm.ErrRecordNotFound) {
        fmt.Println("记录不存在")
    } else if errors.Is(err, jorm.ErrDuplicateKey) {
        fmt.Println("重复的键")
    } else if errors.Is(err, jorm.ErrForeignKey) {
        fmt.Println("外键约束错误")
    } else if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("操作超时")
    } else {
        fmt.Printf("查询错误: %v\n", err)
    }
}

// 常见错误类型
// - jorm.ErrRecordNotFound     记录不存在
// - jorm.ErrInvalidModel       无效的模型
// - jorm.ErrDuplicateKey       重复的键
// - jorm.ErrForeignKey         外键约束
// - jorm.ErrConnectionFailed   连接失败
// - jorm.ErrTransactionAborted 事务终止
// - jorm.ErrInvalidSQL         无效SQL
```

## 多数据库支持

```go
// MySQL
db, _ := jorm.Open("mysql", dsn, nil)

// PostgreSQL
db, _ := jorm.Open("postgres", dsn, nil)

// SQLite
db, _ := jorm.Open("sqlite3", "./test.db", nil)
```

## 最佳实践

### 1. 重用Query对象

```go
// 创建基础查询器
userQuery := db.Model(&User{})

var user1, user2 User
userQuery.Where("id = ?", 1).First(&user1)
userQuery.Where("id = ?", 2).First(&user2)
```

### 2. 批量操作优先

```go
// 推荐方式：批量插入
db.Model(&User{}).BatchInsert(users)

// 不推荐：循环插入
for _, user := range users {
    db.Model(&User{}).Insert(user)
}
```

### 3. 限制查询字段

```go
// 推荐：只查询需要的字段
var users []User
db.Model(&User{}).Select("id", "name").Find(&users)

// 不推荐：查询所有字段
db.Model(&User{}).Find(&users)
```

### 4. 使用索引查询

```go
// 推荐：使用索引字段
var user User
db.Model(&User{}).Where("email = ?", email).First(&user)

// 不推荐：全表扫描
var users []User
db.Model(&User{}).Where("name LIKE ?", "%Alice%").Find(&users)
```

### 5. 使用Context超时控制

```go
// 推荐：为长时间查询设置超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var users []User
db.Model(&User{}).WithContext(ctx).Find(&users)
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/yourusername/jorm"
    _ "github.com/go-sql-driver/mysql"
)

type User struct {
    ID        int64     `jorm:"pk auto"`
    Name      string    `jorm:"size:100 notnull"`
    Email     string    `jorm:"size:100 unique"`
    Age       int       `jorm:"default:0"`
    CreatedAt time.Time `jorm:"auto_time"`
}

func main() {
    // 初始化数据库
    db, err := jorm.Open("mysql", "root:password@tcp(127.0.0.1:3306)/test?charset=utf8mb4", &jorm.Options{
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: time.Hour,
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 自动创建表
    if err := db.AutoMigrate(&User{}); err != nil {
        panic(err)
    }

    // 插入数据
    user := &User{
        Name:  "Alice",
        Email: "alice@example.com",
        Age:   25,
    }
    id, err := db.Model(user).Insert(user)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Inserted user with ID: %d\n", id)

    // 查询数据（带Context）
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    var users []User
    err = db.Model(&User{}).
        WithContext(ctx).
        Where("age > ?", 20).
        OrderBy("created_at DESC").
        Limit(10).
        Find(&users)
    if err != nil {
        panic(err)
    }
    for _, u := range users {
        fmt.Printf("User: %s (%d years old)\n", u.Name, u.Age)
    }

    // 更新数据
    affected, err := db.Model(&User{}).
        Where("id = ?", id).
        Update(map[string]any{"age": 26})
    if err != nil {
        panic(err)
    }
    fmt.Printf("Updated %d rows\n", affected)

    // 使用事务
    err = db.Transaction(func(tx *jorm.Tx) error {
        user2 := &User{Name: "Bob", Email: "bob@example.com", Age: 30}
        _, err := tx.Model(user2).Insert(user2)
        return err // 自动提交或回滚
    })
    if err != nil {
        panic(err)
    }
}
```
