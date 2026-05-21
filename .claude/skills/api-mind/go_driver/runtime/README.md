# apitest — Go 接口自动化运行时

`apitest` 是一个轻量 Go 库，封装了基于 paas-gw 的接口自动化测试主流程：

解析 `.env` 与变量 → 构造 paas-gw 请求 → 发送 → 解析下游响应 → 跑断言 → 抽取变量 → 写 `apitest_<case>.log`。

写完的 `*_test.go` 直接 `go test` 跑，无需任何外部 CLI。

## 1. 何时使用

- 由 `api-test` skill 自动生成接口自动化测试代码（首选）。
- 手写一个一次性的接口验证 / 联调脚本。

不适合写真正的内嵌 unit test —— 它会真的发 HTTP 到 paas-gw，是 *integration test*。

## 2. 目录约定

每个被测方法一个目录，文件命名 `<method>_test.go`：

```
tests/integration/
├── apitest/                              # 本运行时库
└── <method_snake_case>/
    └── <method_snake_case>_test.go
```

Demo：[`tests/integration/get_all_policy_group_meta/`](../get_all_policy_group_meta/)。

## 3. 最小示例

```go
package my_method_test

import (
    "os"
    "testing"

    "<go list -m>/tests/integration/apitest"
)

func TestMyMethod(t *testing.T) {
    token := os.Getenv("APITEST_TOKEN")
    if token == "" {
        t.Skip("APITEST_TOKEN not set")
    }

    suite := apitest.New(t).
        WithEnv(&apitest.EnvConfig{
            PSM:     "tns.tsop.ms_api",
            Env:     "boe_default",
            Branch:  "master",
            Zone:    "China-BOE",
            IDC:     "boe",
            Cluster: "runtime",
        }).
        WithJWTToken(token).
        WithLogDir("./api_test_logs").
        WithGlobalVars(apitest.JSON{
            "tenant_id": "tiktok-test-automation",
        })

    suite.Run(apitest.Case{
        ID:       "TC-01",
        Name:     "happy path",
        Priority: "P0",
        Type:     "RPC",
        Steps: []apitest.Step{{
            Name: "RuntimeGetAllPolicyGroupMeta",
            Type: "RPC",
            API:  "RuntimeGetAllPolicyGroupMeta",
            Body: apitest.JSON{"TenantId": "${{tenant_id}}"},
            Asserts: []string{
                "status_code == 200",
                "typeof $.PolicyGroups == 'list'",
                "len($.PolicyGroups) > 0",
            },
            Extract: map[string]string{
                "first_pg_id": "$.PolicyGroups[0].PolicyGroupId",
            },
        }},
    })
}
```

每个 Step 会变成一个 `t.Run` 子测试 → `go test -v` 输出树结构清晰可读。

## 4. 环境配置

可选 A：`WithEnv(&EnvConfig{...})`（推荐，便于 IDE 跳转/重构）。

可选 B：`WithEnvFile("path/to/.env")`，YAML 格式：

```yaml
psm: tns.tsop.ms_api
env: boe_default
branch: master
zone: China-BOE
idc: boe
cluster: runtime
test_account:
  Cookie: "sessionid=..."
  Hex-Auth-Key: "..."
  Hex-Login-User-Info: "..."
```

`test_account` 下所有键都会作为 HTTP header 注入（RPC step 不注入）。
切勿把真实 token / cookie 提交到仓库，建议放到本地路径并通过 `WithEnvFile(os.Getenv("APITEST_ENV"))` 读。

JWT token 来源：工作流通过 `user_jwt` 获取后设置 `APITEST_TOKEN`，或手动 `export APITEST_TOKEN=<jwt>`；`.env` 不存储 paas-gw JWT。

网关选择由 runtime 的 gateway 层根据 `Zone` / `IDC` / `Env` 自动推导，不需要也不支持在 `.env` 中额外配置域名。默认域名遵循 Explorer OpenAPI 控制面表：CN `https://paas-gw.byted.org/api/v1`，BOE `https://paas-gw-boe.byted.org/api/v1`，BOEI18N/BOETTP `https://paas-gw-boei18n.byted.org/api/v1`，I18N 办公网 `https://bc-useastdt-gw.tiktok-row.net/api/v1`，GCP `https://paas-gw-gcp.tiktoke.org/api/v1`，TTP `https://paas-gw-tx.tiktokd.org/api/v1`，SINF `https://paas-gw.sinf.net/api/v1`。

## 5. 变量与解析

三层作用域，优先级 高 → 低：

1. 上一步 `Extract` 出来的（per-case 内）
2. `Case.Vars`
3. `Suite.WithGlobalVars(...)`

两套占位符语法：

| 语法 | 用途 | 类型行为 |
| --- | --- | --- |
| `${{var}}` | **保留原类型**：单独占位时返回原始 `int / list / map / bool` | 适合放在 JSON Body 里 |
| `${var}` | **始终字符串化**：内嵌到字符串里做拼接 | 适合 URL / header |

举例：

```go
Body: apitest.JSON{
    "Id":   "${{policy_group_id}}", // 保持 int64
    "Path": "/v1/${tenant_id}/list", // 字符串拼接
}
```

## 6. 断言语法

| 写法 | 含义 |
| --- | --- |
| `status_code == 200` | 网关返回 HTTP 状态码 |
| `$.code == 0` | JSONPath 简写，等同 `jsonpath('$.code') == 0` |
| `$.data.items[0].id != null` | 任意比较 |
| `len($.data.items) > 1` | 长度断言 |
| `typeof $.data.id == 'int'` | 类型断言（int / float / string / boolean / list / dict / null） |
| `'admin' in $.data.roles` | 包含断言 |
| `jsonpath('$.code') == 0` | 显式 `jsonpath()` 调用 |

底层使用 [`expr-lang/expr`](https://expr-lang.org)，`$.x.y` 在传给 `expr` 之前会被改写为 `jsonpath('$.x.y')`。

## 7. 抽取语法

```go
Extract: map[string]string{
    "first_id":    "$.data.items[0].id",
    "ticket_id":   "$.data.TicketId",
    "all_ids":     "$.data.items[*].id", // gjson 支持的简单数组通配
}
```

抽出来的值会立即写入 `extracted` 命名空间，下一步可以用 `${{first_id}}` 引用。

## 8. 日志输出

每个 case 写一份 `<logDir>/apitest_<case_id>.log`，分段格式与 `resources/test_report_guide.md` 兼容：

```
===== Step: RuntimeGetAllPolicyGroupMeta =====
--- Request: Business (Curl) ---
curl --location --request POST 'rpc://tns.tsop.ms_api/RuntimeGetAllPolicyGroupMeta' ...

--- Request: Body ---
{ "TenantId": "tiktok-test-automation" }

--- Response: Business (JSON) ---
{ "PolicyGroups": [ ... ] }

--- Metadata: Business ---
Business.StatusCode: 200
Business.LogID: 20260418yyyy

--- Metadata: Gateway ---
Gateway.URL: https://paas-gw-boe.byted.org/api/v1/rpc_request
Gateway.LogID: 20260418xxxx
Gateway.LatencyMs: 142.3
```

未设 `WithLogDir` 时，本地不写日志文件（断言结果仍透到 `t.Errorf` 上）。

## 9. 测试组织建议

- **Priority**：仅做信息标注；想强制顺序就把 P0 的 case 放前面，`go test -v` 按声明顺序跑。
- **stop on failure**：用 `t.FailNow()` / `t.Fatal*` 就能在第一个失败处中断；本库的 `runStep` 故意只调 `t.Errorf`，让多个断言失败都能看到。
- **并行**：默认串行，因为大多数接口测试有共享态。需要并行就在你的 Test 里 `t.Parallel()`。

## 10. 依赖

- `github.com/expr-lang/expr` — 断言表达式求值
- `github.com/tidwall/gjson`  — JSONPath 抽取
- `gopkg.in/yaml.v3`           — `.env` 解析

均已在 `go.mod` 中显式声明。Library 内部只用 `testing.T.Errorf/Fatalf`，没有强依赖 `testify`，但用户在自己的 `*_test.go` 里可以自由 `import testify`。
