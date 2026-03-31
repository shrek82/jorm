# PostgreSQL 连接信息

## 环境信息

| 项目 | 值 |
|------|-----|
| **设备类型** | 公司 MacBook |
| **操作系统** | macOS 15 |
| **PostgreSQL 版本** | 18.3 (通过 Homebrew 安装) |
| **安装方式** | `brew install postgresql@18` |

## 连接信息

| 项目 | 值 |
|------|-----|
| **用户名** | `youxiao` (与 macOS 用户名相同) |
| **密码** | 无 (本地 socket 使用 trust 认证) |
| **默认数据库** | `postgres` |
| **Host** | localhost (或 /tmp/.s.PGSQL.5432 socket) |
| **端口** | 5432 |
| **超级用户** | `youxiao` (已授权 superuser) |

## 连接命令

```bash
# 方式 1：完整命令
psql -U youxiao -d postgres

# 方式 2：简化命令 (使用默认用户)
psql postgres

# 方式 3：指定 host
psql -h localhost -U youxiao -d postgres
```

## 服务管理

```bash
# 启动 PostgreSQL
brew services start postgresql@18

# 停止 PostgreSQL
brew services stop postgresql@18

# 重启 PostgreSQL
brew services restart postgresql@18

# 查看服务状态
brew services list
```

## psql 常用命令

```bash
\l          # 查看所有数据库
\c dbname   # 切换到指定数据库
\du         # 查看所有用户/角色
\dt         # 查看当前数据库的表
\q          # 退出 psql
```

## 数据库列表

| 数据库名 | 用途 | 创建日期 |
|----------|------|----------|
| `postgres` | 默认系统数据库 | - |
| `xy` | 业务数据库 | 2026-03-31 |

## 数据目录

- **数据目录**: `/usr/local/var/postgresql@18`
- **配置文件**: `/usr/local/var/postgresql@18/postgresql.conf`
- **认证配置**: `/usr/local/var/postgresql@18/pg_hba.conf`
