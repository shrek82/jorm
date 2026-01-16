# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-01-16

### 新增 (Added)
- 为 `ValidationErrors` 增加 `First()` 方法，返回第一个验证错误对象。
- 为 `ValidationErrors` 增加 `FirstMsg()` 方法，直接返回第一个错误的字符串消息。
- 新增全局快捷函数 `jorm.FirstMsg(err)`，自动处理错误类型转换，无需手动断言。
- 优化 `ValidationErrors.Error()` 实现，多错误时返回摘要格式（如：`Field: error (and N more errors)`）。
- 优化 `go.mod` 中的 `retract` 指令，合并为版本范围并添加说明注释。
- 新增 LICENSE 文件（MIT），满足 pkg.go.dev 文档索引要求。
- 设置默认日志级别为 `LogLevelError`，减少无关输出。
- 新增灵活字段名匹配：支持大小写不敏感、列名（column tag）和蛇形命名。

### 修复 (Fixed)
- 修复 `validator.go` 中的 `go vet` 错误（非常量格式字符串问题）。
- 修复 `logger.go` 中日志方法命名冲突，将 `log` 重命名为 `emit`。

### 变更 (Changed)
- 重构 `ValidationErrors.Error()` 方法，提升可读性。

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
