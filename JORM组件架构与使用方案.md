# JORM 组件架构与使用方案

## 🏗️ 组件架构设计

### 📦 核心组件接口

```go
// 组件基接口
type Component interface {
    Name() string
    Init(db *DB) error
    Shutdown() error
}

// 结果集定义
type Result struct {
    RowsAffected int64
    LastInsertId int64
    Data         interface{} // 映射后的数据
    Error        error
    RawRows      *sql.Rows   // 原始游标（如果是查询）
}

// 查询中间件接口
type QueryMiddleware interface {
    Component
    Process(ctx context.Context, query *Query, next QueryFunc) (*Result, error)
}

// 数据库事件监听器
type EventListener interface {
    Component
    OnQueryStart(ctx context.Context, query *Query)
    OnQueryEnd(ctx context.Context, query *Query, result *Result, err error)
}
```

### 🎯 组件分类

#### 1. **缓存组件**
```go
type CacheComponent struct {
    Client CacheClient
}

func (c *CacheComponent) Name() string { return "cache" }
func (c *CacheComponent) Init(db *DB) error { return nil }
func (c *CacheComponent) Shutdown() error { return nil }
```

#### 2. **监控组件**
```go
type MetricsComponent struct {
    Reporter MetricReporter
}

func (m *MetricsComponent) Process(ctx context.Context, query *Query, next QueryFunc) (*Result, error) {
    start := time.Now()
    result, err := next(ctx, query)
    duration := time.Since(start)
    
    // 记录指标
    m.Reporter.RecordQuery(query.SQL, duration, err)
    return result, err
}
```

#### 3. **安全组件**
```go
type SecurityComponent struct {
    Validator DataValidator
    Sanitizer SQLSanitizer
}

func (s *SecurityComponent) Process(ctx context.Context, query *Query, next QueryFunc) (*Result, error) {
    // SQL 验证和清理
    sanitizedSQL := s.Sanitizer.Sanitize(query.SQL)
    query.SQL = sanitizedSQL
    
    // 数据验证
    if err := s.Validator.Validate(query.Model); err != nil {
        return nil, err
    }
    
    return next(ctx, query)
}
```

## 📋 组件使用方案

### 🚀 基础使用方式

```go
// 1. 创建数据库实例
db, err := jorm.Open("sqlite3", "test.db")
if err != nil {
    panic(err)
}

// 2. 添加组件
err = db.Use(
    NewRedisCache(WithAddr("localhost:6379")), // 使用 Functional Options
    &Metrics{Reporter: prometheus.NewReporter()},
    &Logger{Level: Info},
)
if err != nil {
    panic(err)
}

// 3. 使用
users := db.Model(&User{}).Cache().Find()
```

### 🎨 高级使用模式

#### 1. **条件性组件启用**
```go
// 生产环境启用监控
if env == "production" {
    db.Use(&Metrics{Reporter: prometheus.NewReporter()})
}

// 开发环境启用详细日志
if env == "development" {
    db.Use(&Logger{Level: Debug})
}
```

#### 2. **组件生命周期管理**
```go
// 组件初始化
func (db *DB) Use(components ...Component) error {
    for _, comp := range components {
        if err := comp.Init(db); err != nil {
            return err
        }
        db.components[comp.Name()] = comp
    }
    return nil
}

// 组件关闭
func (db *DB) Close() error {
    for _, comp := range db.components {
        comp.Shutdown()
    }
    return db.db.Close()
}
```

## 🔄 组件通信机制

### 🔄 中间件链式调用与 Panic 恢复
```go
// 查询处理流程
func (db *DB) executeQuery(ctx context.Context, query *Query) (res *Result, err error) {
    // 增加 Panic 恢复机制
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic in query execution: %v", r)
        }
    }()

    // 构建中间件链
    var next QueryFunc = db.executeRawQuery
    
    // 从后往前构建调用链
    for i := len(db.middlewares) - 1; i >= 0; i-- {
        middleware := db.middlewares[i]
        next = func(ctx context.Context, q *Query) (*Result, error) {
            return middleware.Process(ctx, q, next)
        }
    }
    
    return next(ctx, query)
}
```

### 📡 组件间事件通知
```go
// 事件总线
type EventBus struct {
    subscribers map[string][]EventListener
    mu          sync.RWMutex
}

func (e *EventBus) Publish(event string, data interface{}) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    
    for _, listener := range e.subscribers[event] {
        listener.OnEvent(event, data)
    }
}
```

## 💡 架构优化与改进建议

### 1. 增强配置的类型安全 (Functional Options)

推荐使用 Functional Options 模式来初始化组件，替代直接的结构体赋值，以提供更好的默认值管理和扩展性。

```go
type RedisCache struct {
    addr     string
    password string
    db       int
}

type RedisOption func(*RedisCache)

func WithAddr(addr string) RedisOption {
    return func(c *RedisCache) {
        c.addr = addr
    }
}

func NewRedisCache(opts ...RedisOption) *RedisCache {
    c := &RedisCache{
        addr: "localhost:6379", // 默认值
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### 2. 规范化 Context 管理

组件间传递上下文信息时，应避免使用裸字符串作为 Key。

```go
type ctxKey string
const TraceIDKey ctxKey = "jorm_trace_id"

func WithTraceID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, TraceIDKey, id)
}

func GetTraceID(ctx context.Context) string {
    if v, ok := ctx.Value(TraceIDKey).(string); ok {
        return v
    }
    return ""
}
```

### 3. 依赖管理

建议在 `Component` 接口中增加依赖声明，确保组件初始化的顺序正确。

```go
type Component interface {
    Name() string
    Init(db *DB) error
    Shutdown() error
    Dependencies() []string // 返回依赖的组件名称列表
}
```

## 📈 性能与资源管理

### 🧠 内存优化
```go
// 组件池化管理
type ComponentPool struct {
    components map[string]sync.Pool
    mu         sync.RWMutex
}
// ... (原有代码保持不变)
```
