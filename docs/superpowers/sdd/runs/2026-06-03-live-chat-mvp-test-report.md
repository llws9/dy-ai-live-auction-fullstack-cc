# Live Chat MVP 测试报告

## 基本信息

| 字段 | 值 |
| --- | --- |
| 时间 | `2026-06-03 01:12:47 +0800` |
| 分支 | `feat/live-chat-mvp` |
| Worktree | `/Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat-live-chat-mvp` |
| Commit | `0abb4b5d` |
| 目的 | 重新运行全量测试，确认第二轮 code review 修复后测试用例仍通过 |

## 测试结论

全量自动化测试命令均通过。

| 模块 | 命令 | 结果 |
| --- | --- | --- |
| `backend/auction` | `go test ./...` | PASS |
| `backend/gateway` | `go test ./...` | PASS |
| `backend/product` | `go test ./...` | PASS |
| `backend/test` | `go test ./...` | PASS |
| `backend/seed` | `go test ./...` | PASS |
| `frontend/h5` | `npm test -- --runInBand` | PASS，37 suites / 205 tests |
| `frontend/h5` | `npx tsc --noEmit` | PASS |
| `frontend/admin` | `npm test -- --runInBand` | PASS，4 suites / 42 tests |
| IDE diagnostics | `GetDiagnostics` | PASS，0 diagnostics |

## 构建检查

| 模块 | 命令 | 结果 |
| --- | --- | --- |
| `frontend/h5` | `npm run build` | PASS |
| `frontend/test-dashboard` | `npm ci && npm run build` | PASS |
| `frontend/admin` | `npm run build` | FAIL，既有 TS 严格检查问题，非测试用例失败 |

`frontend/admin` 构建失败的主要类型：

- `@testing-library/jest-dom` matcher 类型未纳入 `tsc` 类型环境，例如 `toBeInTheDocument` / `toBeDisabled`。
- 多个 `pages-new/*` 文件存在未使用 import / 变量。
- 若干 UI prop 类型不匹配，例如 `Badge` variant 字符串收窄类型。
- `src/shared/api/index.ts` 重新导出 `request`，但 `request.ts` 未导出该 symbol。

## 本轮补充修复

全量验证过程中发现并修复了两个会阻塞测试执行的问题：

| 问题 | 根因 | 修复 |
| --- | --- | --- |
| `backend/seed go test ./...` 要求 `go mod tidy`，随后编译失败 | `product-service/model.Order.FinalPrice` 已改为 `decimal.Decimal`，seed 仍传 `float64`；同时 `log.Printf` 参数数量不匹配触发 vet | `go mod tidy`；seed 订单金额改用 `decimal.NewFromInt`；补齐 summary 日志中的总数参数 |
| `frontend/admin npm test` 无法加载 Jest 配置 | `package.json` 使用 `"type":"module"`，但 `jest.config.js` 使用 CommonJS `module.exports` | 新增 `jest.config.cjs`，`test` / `test:coverage` script 显式指定 CJS 配置 |

## 运行备注

- `frontend/h5` Jest 输出包含预期内 console noise：`useTheme` 抛错测试、React Router future flag warning、重连上限日志等；退出码为 0，测试全部通过。
- `frontend/h5 npm run build` 输出 Vite CJS Node API deprecation warning；退出码为 0。
- `frontend/test-dashboard npm ci` 输出 2 个 moderate audit vulnerabilities；未执行自动修复，避免引入破坏性依赖升级。
- `frontend/test-dashboard npm run build` 输出 chunk size warning；退出码为 0。

## 当前未提交变更

```text
M  backend/seed/generators.go
M  backend/seed/go.mod
M  backend/seed/go.sum
M  backend/seed/main.go
D  frontend/admin/jest.config.js
M  frontend/admin/package.json
?? frontend/admin/jest.config.cjs
?? docs/superpowers/sdd/runs/2026-06-03-live-chat-mvp-test-report.md
```

建议后续提交为：

```text
test: restore full verification matrix after live chat fixes
```
