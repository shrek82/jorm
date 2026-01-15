---
name: "code-commenter"
description: "Adds detailed comments to code files. Invoke when user asks to add comments, document code, or explain code. RULES: Core methods=English, Tests=Chinese."
---

# Code Commenter Skill

This skill specializes in adding meaningful, high-quality comments to source code, with specific language rules for production vs. test code.

## Goals
1. **Documentation**: Ensure core and main methods have proper documentation.
2. **Clarification**: Add inline comments to explain complex logic in core methods.
3. **Test Context**: clearly explain test scenarios and verification steps.

## Guidelines

### 1. Production Code (Core & Main Methods) -> ENGLISH
- **Target**: ONLY add comments to **Core Methods** (critical logic, interfaces, complex algorithms) and **Main Methods** (entry points, public APIs).
- **Language**: **ENGLISH ONLY**.
- **Style**:
  - Exported methods: GoDoc style (e.g., `// FunctionName does X...`).
  - Complex logic: Concise inline English explanations.
- **Skip**: Trivial getters/setters, simple logic, or boilerplate.

### 2. Test Code (Test Cases) -> CHINESE
- **Target**: **Every** added or modified test case/function.
- **Language**: **CHINESE ONLY**.
- **Content**:
  - Explain the **Scenario** (测试场景).
  - Explain the **Verification Logic** (验证逻辑).
  - Explain any **Mock Data** purpose (模拟数据用途).
- **Example**:
  ```go
  // TestUserLogin_Success 测试用户成功登录的场景
  // 步骤：
  // 1. 模拟数据库返回有效用户
  // 2. 调用 Login 方法
  // 3. 验证返回的 Token 不为空
  func TestUserLogin_Success(t *testing.T) { ... }
  ```

### 3. General Rules
- **No Logic Changes**: strictly ONLY add comments. Do not modify the code structure or logic.
- **Conciseness**: Avoid redundant comments (e.g., `i++ // increment i`).

### Process
1. **Identify Type**: Is this Production Code or Test Code?
2. **Select Language**: English for Production (Core/Main), Chinese for Tests.
3. **Filter**: For Production, only target Core/Main methods. For Tests, target all modified cases.
4. **Apply**: Insert comments.

## Example Interaction

**User**: "Add comments to this PR."
**Agent**:
- Finds `core/user.go`: Adds **English** comments to `Login` and `Register` (Core methods).
- Finds `core/user_test.go`: Adds **Chinese** comments to `TestLogin` explaining the test steps.
