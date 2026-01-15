# JORM 验证器设计文档 (Validator Design)

本文档详细说明了 JORM 验证器系统的设计，旨在为数据模型提供灵活、可扩展的校验机制。

## 1. 核心目标

- 支持在 `Insert` 和 `Update` 操作前进行数据校验。
- 支持为同一个 Model 定义多个不同场景的验证器（如 `Default`, `Admin`, `Strict`）。
- 提供内置规则（非空、长度限制、正则等）并支持自定义逻辑。
- 支持条件校验（如果字段不为零值，则执行特定规则）。

## 2. API 设计

### 2.1 新增方法

在 `Query` 结构体中新增以下方法：

```go
// InsertWithValidator 使用指定的验证器进行插入
func (q *Query) InsertWithValidator(value any, validators ...Validator) (int64, error)

// UpdateWithValidator 使用指定的验证器进行更新
func (q *Query) UpdateWithValidator(value any, validators ...Validator) (int64, error)

// Validate 独立验证方法，不触发数据库操作
func Validate(value any, validators ...Validator) error
```

### 2.2 验证器定义

验证器是一个接受数据对象并返回错误的函数：

```go
type Validator func(value any) error
```

为了支持一次性返回多个字段的错误，JORM 内部定义了 `ValidationErrors`：

```go
// ValidationErrors 包含多个字段的错误信息
type ValidationErrors map[string][]error

func (v ValidationErrors) Error() string {
    // 自动格式化所有错误消息
}
```
 JORM 提供 `jorm.Rules` 类型，通过 **Map 映射** 的方式定义字段规则，配合链式属性，使代码达到极致简洁：

```go
// 语法结构
jorm.Rules{
    "字段名": {规则1, 规则2.Msg("错误提示"), ...},
}
```

## 3. 使用方式

### 3.1 在 Model 中定义验证方法

建议在 Model 中定义返回 `Validator` 的方法：

```go
func (m *User) CommonValidator() jorm.Validator {
    return jorm.Rules{
        "Name": {jorm.Required.Msg("用户姓名不能为空"), jorm.MinLen(2), jorm.MaxLen(50)},
        "Age":  {jorm.Range(0, 150)},
    }
}

func (m *User) AdminValidator() jorm.Validator {
    return func(value any) error {
        user := value.(*User)
        if user.Role != "admin" {
            return errors.New("只有管理员可以操作")
        }
        return nil
    }
}
```

### 3.2 调用示例

你可以直接将方法调用作为参数传递，支持组合多个验证器：

```go
user := &User{Name: "Alice", Age: 25}

// 同时使用通用验证和管理员验证
db.Model(user).InsertWithValidator(user, 
    user.CommonValidator(), 
    user.AdminValidator(),
)
```

### 3.3 独立验证 (Standalone Validation)

如果你只想执行校验逻辑而不进行数据库操作（例如在 Web 层接收到参数后立即校验），可以使用 `jorm.Validate`：

```go
user := &User{Name: "A", Age: -1}

// 独立执行校验
if err := jorm.Validate(user, user.CommonValidator()); err != nil {
    fmt.Println("数据非法:", err)
    return
}
```

## 4. 验证规则定义 (Concise Rules)

### 4.1 基础校验
| 规则 | 说明 | 示例 |
| :--- | :--- | :--- |
| `Required` | 字段不能为空值（零值） | `"Name": {jorm.Required}` |
| `MaxLen(n)` | 长度不能大于 n | `"Bio": {jorm.MaxLen(200)}` |
| `MinLen(n)` | 长度不能小于 n | `"Pwd": {jorm.MinLen(8)}` |
| `Range(min, max)` | 数值必须在 [min, max] 范围内 | `"Age": {jorm.Range(1, 100)}` |
| `In(v...)` | 值必须在指定的枚举列表中 | `"Status": {jorm.In(1, 2, 3)}` |

### 4.2 格式校验
| 规则 | 说明 | 示例 |
| :--- | :--- | :--- |
| `Email` | 必须是有效的 Email 格式 | `"Email": {jorm.Email}` |
| `Mobile` | 必须是有效的手机号格式 | `"Phone": {jorm.Mobile}` |
| `URL` | 必须是有效的 URL 地址 | `"Link": {jorm.URL}` |
| `IP` | 必须是有效的 IPv4 或 IPv6 地址 | `"LastIP": {jorm.IP}` |
| `JSON` | 必须是有效的 JSON 字符串 | `"Config": {jorm.JSON}` |
| `UUID` | 必须是有效的 UUID 格式 | `"TraceID": {jorm.UUID}` |
| `Datetime(fmt)` | 必须符合指定的日期时间格式 | `"Day": {jorm.Datetime("2006-01-02")}` |
| `Regexp(p)` | 必须匹配指定的正则表达式 | `"SN": {jorm.Regexp("^SN-\\d+$")}` |

### 4.3 字符集与内容校验
| 规则 | 说明 | 示例 |
| :--- | :--- | :--- |
| `Numeric` | 仅允许数字字符 | `"Zip": {jorm.Numeric}` |
| `Alpha` | 仅允许英文字母 | `"Code": {jorm.Alpha}` |
| `AlphaNumeric` | 允许英文字母和数字 | `"ID": {jorm.AlphaNumeric}` |
| `Contains(s)` | 必须包含指定的子字符串 | `"Tags": {jorm.Contains("go")}` |
| `Excludes(s)` | 不能包含指定的子字符串 | `"Note": {jorm.Excludes("badword")}` |
| `NoHTML` | 禁止包含任何 HTML 标签 | `"Bio": {jorm.NoHTML}` |

**所有规则对象**（无论是基础校验还是格式校验）都实现了统一的接口，支持以下链式调用：

| 修饰符 | 说明 | 示例 |
| :--- | :--- | :--- |
| `.Msg(string)` | **错误定制**：自定义该规则失败时的错误消息。 | `jorm.MinLen(5).Msg("太短了")` |
| `.Optional()` | **可选触发**：仅在字段非零值时执行。**适用于任何规则**。 | `jorm.Range(1, 10).Optional()` |
| `.When(fn)` | **自定义条件**：仅当 `fn` 返回 `true` 时执行。 | `jorm.Required.When(isStrict)` |

#### 示例：全类型支持 Optional
`Optional()` 不仅限于手机号或邮箱，它可以用于任何校验逻辑：
```go
jorm.Rules{
    "Age":    {jorm.Range(18, 60).Optional()}, // 如果填了年龄，必须在18-60之间
    "Score":  {jorm.Min(60).Optional().Msg("及格分至少60")}, // 如果填了分数，必须及格
    "Bio":    {jorm.MaxLen(100).Optional()}, // 如果填了简介，长度不能超标
}
```

### 4.4 字符集校验
| 规则 | 说明 | 示例 |
| :--- | :--- | :--- |
| `Numeric()` | 仅允许数字字符 | `Field("Zip").Numeric()` |
| `Alpha()` | 仅允许英文字母 | `Field("Code").Alpha()` |
| `AlphaNumeric()` | 允许英文字母和数字 | `Field("ID").AlphaNumeric()` |

## 5. 综合示例 (Full Example)

以下是一个完整的业务场景示例，展示了如何定义和使用验证器：

```go
type User struct {
    ID    int64
    Name  string
    Email string
    Age   int
    Role  string
}

// CommonValidator 定义基础校验规则
func (m *User) CommonValidator() jorm.Validator {
    return jorm.Rules{
        "Name": {
            jorm.Required.Msg("用户姓名不能为空"),
            jorm.MinLen(2), jorm.MaxLen(20),
        },
        "Email": {
            jorm.Required,
            jorm.Email.Msg("邮箱格式不正确"),
        },
        "Age": {
            jorm.Range(18, 120).Msg("用户必须成年且年龄合法"),
        },
        "Phone": {
            jorm.Mobile.Optional().Msg("手机号格式不正确"),
        },
    }
}

// AdminValidator 专门用于管理员操作的校验
func (m *User) AdminValidator() jorm.Validator {
    return func(value any) error {
        u := value.(*User)
        if u.Role != "admin" {
            return errors.New("操作失败：当前用户权限不足")
        }
        return nil
    }
}

func main() {
    db, _ := jorm.Open(...)
    user := &User{
        Name:  "Shrek",
        Email: "invalid-email",
        Age:   10,
        Role:  "guest",
    }

    // 1. 基础插入校验
    // 会报错: "邮箱格式不正确" (Email 校验失败) 或 "用户必须成年" (Age 校验失败)
    _, err := db.Model(user).InsertWithValidator(user, user.CommonValidator())
    if err != nil {
        fmt.Println("校验失败:", err)
    }

    // 2. 组合校验 (基础校验 + 管理员校验)
    // 即使基础校验通过，也会因为 Role != "admin" 被 AdminValidator 拦截
    _, err = db.Model(user).UpdateWithValidator(user, 
        user.CommonValidator(), 
        user.AdminValidator(),
    )
    if err != nil {
        fmt.Println("组合校验失败:", err)
    }

    // 3. 独立校验示例 (不操作数据库)
    // 场景：在逻辑处理前先检查数据合法性
    newUser := &User{Name: "Bob"}
    if err := jorm.Validate(newUser, newUser.CommonValidator()); err != nil {
        fmt.Printf("独立校验未通过: %v\n", err)
    }
}
```

## 7. 进阶特性 (Advanced)

### 7.1 跨字段校验 (Cross-field)

你可以通过自定义验证器实现复杂的跨字段逻辑：

```go
func (m *User) PasswordMatchValidator() jorm.Validator {
    return func(value any) error {
        u := value.(*User)
        if u.Password != u.ConfirmPassword {
            return errors.New("两次输入的密码不一致")
        }
        return nil
    }
}
```
