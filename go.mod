module github.com/shrek82/jorm

go 1.25.3

retract (
	v0.1.6-0.20260112163748-1c18c100b4ab
	// 这些早期版本包含不稳定的 API 且缺少必要的开源许可证
	[v0.1.0, v0.1.5]
	[v0.1.0-beta.1, v0.1.0-beta.2]
	v0.0.0-20260111101838-508554935029
)

require (
	github.com/go-sql-driver/mysql v1.9.3
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/redis/go-redis/v9 v9.17.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)
