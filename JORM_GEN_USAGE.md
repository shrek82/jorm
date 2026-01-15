# JORM-GEN 使用指南

`jorm-gen` 是一个为 JORM 框架设计的自动化模型生成工具。它可以根据现有的数据库表结构，自动生成对应的 Go 语言结构体代码，并带有完整的 `jorm` 标签。

## 1. 安装

确保你的机器上已安装 Go (1.25.3+)，然后运行以下命令进行安装：

```bash
go install github.com/shrek82/jorm/cmd/jorm-gen@latest
```

安装完成后，你可以直接在终端输入 `jorm-gen` 来验证是否安装成功。

## 2. 命令行参数

| 参数 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `-driver` | `sqlite3` | 数据库驱动类型，可选：`sqlite3`, `mysql`, `postgres` |
| `-dsn` | (必填) | 数据库连接字符串 (Data Source Name) |
| `-table` | `""` | 指定要生成的表名。如果为空，则生成数据库中所有的表 |
| `-pkg` | `models` | 生成的 Go 代码包名 |
| `-out` | `./models` | 代码输出目录，如果目录不存在会自动创建 |
| `-overwrite`| `false` | 如果目标文件已存在，是否覆盖。默认跳过已存在文件 |

## 3. 使用示例

### SQLite
生成当前目录下 `test.db` 中的所有表：
```bash
jorm-gen -driver sqlite3 -dsn "./test.db" -out "./internal/models"
```

### MySQL
生成指定数据库中的所有表：
```bash
jorm-gen -driver mysql -dsn "user:password@tcp(127.0.0.1:3306)/dbname" -pkg mymodels
```

### PostgreSQL
生成指定表并强制覆盖现有文件：
```bash
jorm-gen -driver postgres -dsn "host=localhost user=postgres password=secret dbname=mydb sslmode=disable" -table users -overwrite
```

## 4. 生成的代码特性

`jorm-gen` 会根据数据库字段自动推断并生成以下特性：

- **命名转换**：数据库的 `snake_case` 命名会自动转换为 Go 的 `PascalCase`。特别地，`id` 会转换为 `ID`。
- **字段标签生成**：
    - `column:xxx`: 指定列名。
    - `pk`: 标识主键。
    - `auto`: 标识自增字段。
    - `notnull`: 标识非空约束。
    - `unique`: 标识唯一索引。
    - `default:xxx`: 标识默认值。
    - `size:xxx`: 标识字段长度（如 VARCHAR(100)）。
    - `auto_time`: 针对 `created_at` 字段自动添加。
    - `auto_update`: 针对 `updated_at` 字段自动添加。
- **注释支持**：自动从数据库（MySQL/Postgres）提取字段注释并生成 Go 注释。
- **类型映射**：智能将数据库类型映射为 Go 标准类型（int64, string, time.Time 等）。
- **TableName 方法**：自动生成 `TableName()` 方法以支持自定义表名。

## 5. 注意事项

1. **驱动依赖**：本工具内置了 `sqlite3`, `mysql`, `postgres` 驱动。
2. **PostgreSQL 模式**：目前默认扫描 `public` schema 下的表。
3. **主键检测**：在 MySQL 和 SQLite 中可以精确识别自增主键。在 PostgreSQL 中，如果字段名被标记为主键约束，工具会将其标记为 `pk`。
