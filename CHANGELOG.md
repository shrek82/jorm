# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-01-16 16:30

### 新增 (Added)
- 新增 `TimeScanner` 结构体 (core/query.go)，用于自动处理 MySQL 的日期时间扫描。
- 支持将 `[]byte` 和字符串格式的日期自动转换为 `time.Time`。
- 支持自动处理 MySQL 的 `0000-00-00` 无效日期值为零值。

### 修复 (Fixed)
- 修复了 MySQL 驱动返回 `[]uint8` 类型导致 `scanRow` 无法转换到 `*time.Time` 的 panic 问题。
- 修复了 MySQL 方言中 `size`, `notnull`, `default` 标签被忽略的问题，现在能正确生成 `varchar(N)`, `NOT NULL` 和 `DEFAULT` SQL。
- 修复了 `logger/logger.go` 中 SQL 日志格式化参数命名冲突导致的编译错误。
- 修复了 `Preload` 关联加载时 `relation` 标签解析错误导致的测试失败。

### 变更 (Changed)
- 优化了 `core/preload.go` 中的扫描逻辑，集成 `TimeScanner` 以增强健壮性。
