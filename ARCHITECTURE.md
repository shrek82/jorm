# github.com/shrek82/jorm 架构设计

## 设计理念

- **轻量**：核心依赖最少，代码精简
- **高性能**：优化反射使用，缓存预编译SQL，批量操作优化
- **可链式操作**：流畅的API设计
- **灵活性**：基于反射的动态类型处理
- **职责清晰**：模块化分层设计
- **兼容性**：支持Go 1.13+

## 核心架构

```
github.com/shrek82/jorm/
├── core/
│   ├── db.go          # DB主入口，连接池管理
│   ├── query.go       # Query查询构建器，链式操作起点
│   ├── builder.go     # SQL构建器接口和实现
│   ├── tx.go          # Transaction事务
│   └── driver.go      # 数据库驱动抽象
├── query/
│   ├── where.go       # WHERE条件构建
│   ├── order.go       # ORDER BY构建
│   ├── limit.go       # LIMIT/OFFSET构建
│   └── join.go        # JOIN操作构建
├── model/
│   ├── model.go       # 模型定义和元数据
│   ├── tag.go         # 标签解析
│   └── field.go       # 字段映射
├── dialect/
│   ├── dialect.go     # 方言接口
│   ├── mysql.go       # MySQL实现
│   ├── postgres.go    # PostgreSQL实现
│   └── sqlite.go      # SQLite实现
├── pool/
│   ├── pool.go        # 连接池接口
│   └── stdpool.go     # 标准库连接池实现
└── logger/
    ├── logger.go      # 日志接口
    └── stdlog.go      # 标准日志实现
```

## 核心组件

### 1. DB - 数据库主入口

```go
type DB struct {
    dialect   dialect.Dialect
    pool      pool.Pool
    logger    logger.Logger
    prepared  sync.Map  // 预编译SQL缓存
}

type Options struct {
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    LogLevel        LogLevel
}
```

**职责**：
- 管理数据库连接池
- 提供Model查询创建
- 缓存预编译SQL语句
- 全局配置管理

### 2. Query - 查询构建器

```go
type Executor interface {
    QueryContext(ctx context.Context, sql string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, sql string, args ...any) (sql.Result, error)
}

type Query struct {
    executor  Executor
    builder   Builder
    ctx       context.Context
}

func (db *DB) Model(value any) *Query
func (db *DB) Table(name string) *Query
```

**职责**：
- 链式查询的上下文
- 管理SQL构建器
- 执行CRUD操作
- 通过反射处理模型绑定
- 支持超时和取消操作

### 3. Tx - 事务对象

```go
type Tx struct {
    db      *DB
    tx      *sql.Tx
}

func (db *DB) Begin() (*Tx, error)
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error)
func (db *DB) Transaction(fn func(*Tx) error) error
func (tx *Tx) Model(value any) *Query
func (tx *Tx) Table(name string) *Query
func (tx *Tx) Commit() error
func (tx *Tx) Rollback() error
```

**职责**：
- 管理事务生命周期
- 提供事务内查询
- 提交/回滚操作
- 自动错误处理和回滚

### 3. Builder - SQL构建器

```go
type Builder interface {
    // 条件构建
    Where(condition string, args ...any) Builder
    OrWhere(condition string, args ...any) Builder
    WhereIn(column string, values any) Builder
    
    // 查询构建
    Select(columns ...string) Builder
    Join(table, on, joinType string) Builder
    OrderBy(column string) Builder
    Limit(n int) Builder
    Offset(n int) Builder
    
    // 生成SQL
    Build() (sql string, args []any)
}

type SQLBuilder struct {
    dialect   dialect.Dialect
    tableName string
    clauses   []*Clause
}

type Clause struct {
    Type  ClauseType  // SELECT, WHERE, ORDER, etc.
    Value any
}
```

**职责**：
- 构建SQL语句
- 支持链式调用
- 处理方言差异
- 参数化查询

### 4. Model - 模型元数据

```go
type Model struct {
    TableName string
    Fields    []*Field
    PKField   *Field
}

type Field struct {
    Name      string  // 结构体字段名
    Column    string  // 数据库列名
    Type      reflect.Type
    IsPK      bool
    IsAuto    bool
    Tag       *Tag
}
```

**职责**：
- 解析模型结构体
- 缓存字段映射
- 提供表信息
- 标签解析

### 5. Dialect - 方言接口

```go
type Dialect interface {
    DataTypeOf(reflect.Type) string
    Quote(string) string
    HasTable(*DB, string) (bool, error)
    CreateTableSQL(*model.Model) (string, []any)
    InsertSQL(string, []string, int) (string, []any)
}

type BaseDialect struct {
    quoteChar byte
}
```

**职责**：
- 处理数据库方言差异
- 生成特定数据库的SQL
- 类型映射

## 链式操作实现

```go
// 基于反射的查询链
func (q *Query) Where(cond string, args ...any) *Query {
    q.builder.Where(cond, args...)
    return q
}

func (q *Query) OrderBy(field string) *Query {
    q.builder.OrderBy(field)
    return q
}

func (q *Query) Limit(n int) *Query {
    q.builder.Limit(n)
    return q
}

func (q *Query) WithContext(ctx context.Context) *Query {
    q.ctx = ctx
    return q
}

// 查询结果需要传入目标变量
func (q *Query) Find(dest any) error {
    return q.FindContext(q.ctx, dest)
}

func (q *Query) FindContext(ctx context.Context, dest any) error {
    sql, args := q.builder.Build()
    rows, err := q.executor.QueryContext(ctx, sql, args...)
    if err != nil {
        return err
    }
    defer rows.Close()
    return q.scanInto(rows, dest)
}

func (q *Query) First(dest any) error {
    q.builder.Limit(1)
    sql, args := q.builder.Build()
    rows, err := q.executor.QueryContext(q.ctx, sql, args...)
    if err != nil {
        return err
    }
    defer rows.Close()
    return q.scanOne(rows, dest)
}
```

## 反射驱动设计

使用反射实现灵活的类型处理：

```go
// 创建查询（基于模型实例）
func (db *DB) Model(value any) *Query

// 创建查询（基于表名）
func (db *DB) Table(name string) *Query

// 查询单条（传入接收变量）
func (q *Query) First(dest any) error

// 查询多条（传入切片指针）
func (q *Query) Find(dest any) error

// 插入（自动填充主键）
func (q *Query) Insert(value any) (int64, error)

// 更新（支持结构体或map）
func (q *Query) Update(value any) (int64, error)

// 删除
func (q *Query) Delete() (int64, error)
```

## 性能优化策略

### 1. 模型元数据缓存

```go
// 全局缓存，避免重复反射解析
var modelCache sync.Map

func getModel(value any) (*model.Model, error) {
    typ := reflect.TypeOf(value)
    if typ.Kind() == reflect.Ptr {
        typ = typ.Elem()
    }
    
    // 使用类型全名作为键
    key := typ.String()
    
    // 从缓存读取
    if cached, ok := modelCache.Load(key); ok {
        return cached.(*model.Model), nil
    }
    
    // 解析并缓存（只执行一次）
    m, err := parseModel(value)
    if err != nil {
        return nil, err
    }
    
    modelCache.Store(key, m)
    return m, nil
}
```

### 2. 字段映射缓存

```go
// 缓存字段访问器，加速数据扫描
type FieldMapper struct {
    indexes map[string]int           // 列名 -> 字段索引
    setters map[int]func(v any) error // 字段赋值函数
}

var fieldMapperCache sync.Map
```

### 3. SQL预编译缓存

```go
// 缓存预编译语句，减少编译开销
type StmtCache struct {
    cache sync.Map
}

func (c *StmtCache) Get(key string) (*sql.Stmt, bool) {
    if stmt, ok := c.cache.Load(key); ok {
        return stmt.(*sql.Stmt), true
    }
    return nil, false
}

func (c *StmtCache) Set(key string, stmt *sql.Stmt) {
    c.cache.Store(key, stmt)
}
```

### 4. 批量操作优化

```go
func (q *Query) BatchInsert(value any) (int64, error) {
    // 检查是否为切片
    val := reflect.ValueOf(value)
    if val.Kind() != reflect.Slice {
        return 0, ErrInvalidModel
    }
    
    if val.Len() == 0 {
        return 0, nil
    }
    
    // 构建批量插入SQL
    sql, args := q.buildBatchInsert(value)
    result, err := q.executor.ExecContext(q.ctx, sql, args...)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}
```

## 职责分离

| 组件 | 职责 | 不负责 |
|------|------|--------|
| DB | 连接管理、SQL缓存、Model创建 | 具体查询、SQL构建 |
| Query | 链式查询上下文、CRUD操作 | SQL生成、数据解析 |
| Tx | 事务管理、事务内查询 | SQL生成、业务逻辑 |
| Builder | SQL语句构建 | 数据库操作、模型解析 |
| Model | 元数据管理 | SQL生成、数据库连接 |
| Dialect | 方言适配 | 查询逻辑、模型管理 |
| Pool | 连接池管理 | SQL构建、业务逻辑 |

## 扩展点

1. **自定义驱动**：实现driver.Driver接口
2. **自定义日志**：实现logger.Logger接口
3. **自定义方言**：实现dialect.Dialect接口
4. **钩子函数**：Before/After操作钩子
5. **中间件**：查询中间件链

## 钩子接口

```go
// 插入钩子
type BeforeInserter interface {
    BeforeInsert() error
}
type AfterInserter interface {
    AfterInsert(id int64) error
}

// 更新钩子
type BeforeUpdater interface {
    BeforeUpdate() error
}
type AfterUpdater interface {
    AfterUpdate() error
}

// 删除钩子
type BeforeDeleter interface {
    BeforeDelete() error
}
type AfterDeleter interface {
    AfterDelete() error
}

// 查询钩子
type AfterFinder interface {
    AfterFind() error
}
```

## 错误定义

```go
// 结构化错误，便于判断和扩展
type JormError struct {
    Code    int
    Message string
    Err     error
}

func (e *JormError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *JormError) Unwrap() error { return e.Err }

const (
    ErrCodeNotFound = 1
    ErrCodeInvalidModel = 2
    ErrCodeDuplicateKey = 3
    ErrCodeForeignKey = 4
    ErrCodeConnectionFailed = 5
    ErrCodeTransactionAborted = 6
    ErrCodeInvalidSQL = 7
)

var (
    ErrRecordNotFound     = &JormError{Code: ErrCodeNotFound, Message: "record not found"}
    ErrInvalidModel       = &JormError{Code: ErrCodeInvalidModel, Message: "invalid model"}
    ErrDuplicateKey       = &JormError{Code: ErrCodeDuplicateKey, Message: "duplicate key"}
    ErrForeignKey         = &JormError{Code: ErrCodeForeignKey, Message: "foreign key constraint"}
    ErrConnectionFailed   = &JormError{Code: ErrCodeConnectionFailed, Message: "connection failed"}
    ErrTransactionAborted = &JormError{Code: ErrCodeTransactionAborted, Message: "transaction aborted"}
    ErrInvalidSQL         = &JormError{Code: ErrCodeInvalidSQL, Message: "invalid sql"}
)
```

## 依赖管理

核心依赖：
- `database/sql` - 标准数据库接口
- `context` - 上下文支持
- `reflect` - 反射（最小化使用）
- `sync` - 并发控制
 - `fmt` - 错误消息格式化（文档中示例）

可选依赖：
- 驱动：`github.com/go-sql-driver/mysql`等
- 日志：`logrus`、`zap`等

## 版本策略

- 支持 Go 1.13+ （基于反射设计）
- 向后兼容的API设计
- 语义化版本控制
