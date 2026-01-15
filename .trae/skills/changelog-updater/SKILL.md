---
name: "changelog-updater"
description: "Updates CHANGELOG.md with recent changes. RULES: MUST use CHINESE, MUST include Date & Time (YYYY-MM-DD HH:mm)."
---

# Changelog Updater Skill

This skill manages the `CHANGELOG.md` file, ensuring all project changes are recorded in a standardized format.

## Standards
- **Format**: Follow [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) conventions.
- **Date Format**: `YYYY-MM-DD HH:mm` (Precision to minute).
- **Language**: **CHINESE ONLY** (Must use Chinese for all descriptions).
- **Grouping**: Group changes under these subsections:
  - `### 新增 (Added)`
  - `### 变更 (Changed)`
  - `### 废弃 (Deprecated)`
  - `### 移除 (Removed)`
  - `### 修复 (Fixed)`
  - `### 安全 (Security)`

## Workflow
1. **Check Existence**: Check if `CHANGELOG.md` exists. If not, create it with a standard header.
2. **Analyze Context**: Review recent code changes or user input to identify what to log.
3. **Format Entry**: Create a new entry section with the current timestamp.
   - Example header: `## [Unreleased] - 2025-01-16 14:30`
4. **Append**: Add the entry to the file, maintaining chronological order (newest on top).

## Example Structure

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased] - 2025-01-16 14:30

### 新增 (Added)
- 新增 `TimeScanner` 用于处理 MySQL 日期解析。

### 修复 (Fixed)
- 修复 `scanRow` 在处理 `[]byte` 转 `time.Time` 时的 panic 问题。
```

## Rules
- **Mandatory Chinese**: All content descriptions must be in Chinese.
- **Mandatory Timestamp**: Every update block must carry a `YYYY-MM-DD HH:mm` timestamp.
- **Clarity**: Write clear, concise summaries.
- **No Duplicates**: Check if the entry already exists before adding.
