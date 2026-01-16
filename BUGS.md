# JORM 已知 BUG 清单

## 测试失败
- **TestIntegration/MultipleWhereAndSelect**: 此测试失败，返回了错误的邮箱值（位于 `/Users/up/projects/jorm/tests/integration_test.go` 第 599 行）
  - 预期值: "multi@example.com"
  - 实际值: 空字符串

## 代码中发现的潜在问题
- **调试语句**: 在 `/Users/up/projects/jorm/tests/debug_test.go:59` 中发现了调试打印语句
- **潜在未处理的错误情况**: 在整个代码库中发现了多个可能需要审查的错误处理路径

## 需要调查的区域
- 带有多个 WHERE 和 SELECT 子句的查询构建可能存在字段映射问题
- 预加载功能在关系加载方面可能存在边界情况
- 扫描器功能应审查以确保适当的错误处理