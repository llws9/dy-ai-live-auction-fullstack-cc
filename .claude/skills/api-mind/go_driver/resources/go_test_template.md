# Go Test File Template

This template is the **canonical shape** for the `*_test.go` files emitted by the api-test skill.

It runs against the in-repo `apitest` Go runtime (`{{APITEST_IMPORT_PATH}}`, resolved by `go list -m` + `/tests/integration/apitest`), which sends requests through `paas-gw` and writes per-case `apitest_<case_id>.log` files consumed by `resources/test_report_guide.md`.

## 1. File Location

```
<REPO_ROOT>/tests/integration/<METHOD_SNAKE_CASE>/<METHOD_SNAKE_CASE>_test.go
```

- One file per **target method** (HTTP path or RPC func).
- Package name: `<METHOD_SNAKE_CASE>_test`.
- Test func name: `Test<MethodPascalCase>`.

## 2. Skeleton

The generated style intentionally follows Tesla-Go's readable shape:
`TestXxx(t)` → context → request → call → assert.

`{{REQUEST_TYPE}}` is `HTTP` or `RPC`, chosen per the target interface type. `{{REQUEST_METHOD_FIELD}}` expands differently for each:

| `{{REQUEST_TYPE}}` | `{{REQUEST_METHOD_FIELD}}` expansion | Example |
|---|---|---|
| `HTTP` | `Method: "<VERB>",`<br>`Path: "<path>",` | `Method: "POST",`<br>`Path: "/api/v2/example/detail",` |
| `RPC` | `Method: "<RPCFuncName>",` | `Method: "SearchBank",` |

```go
package {{METHOD_SNAKE}}_test

import (
	"testing"

	"{{APITEST_IMPORT_PATH}}"
)

func Test{{METHOD_PASCAL}}(t *testing.T) {
	env := apitest.EnvFromFile(t)
	ctx := apitest.NewContext(t, env).WithCaseID("{{CASE_ID}}")

	req := apitest.{{REQUEST_TYPE}}Request{
		{{REQUEST_METHOD_FIELD}}
		Body: apitest.JSON{
			// Built per SKILL.md §2.2 请求数据构造策略.
		},
		Extract: map[string]string{
			// "var_name": "$.jsonpath",   // only when case.md has Extract column
		},
	}

	resp := apitest.Call{{REQUEST_TYPE}}(ctx, req)

	// Use extracted values in subsequent steps (multi-step scenarios):
	// val := resp.ExtractString("var_name")
	// num := resp.ExtractInt64("var_name")

	apitest.Assert(t, resp,
		"status_code == 200",
		"$.code == 0",
	)
}
```

## 3. Per-Case Block (one per `case.md` row)

For every test case in `case.md` emit one ordinary `TestXxx(t)` function. Add a
one-line comment immediately above each generated test function in the format
`// TC-xxx: <case.md 测试场景>` so reviewers can understand the case intent without
opening `case.md`. Keep the body flat: build request, call gateway, then assert.
Multi-step scenarios use multiple `CallHTTP` / `CallRPC` calls and explicit Go variables.

**HTTP example:**

```go
resp := apitest.CallHTTP(ctx, apitest.HTTPRequest{
	Method: "GET",
	Path:   "/api/v2/example/detail",
	Params: map[string]string{
		"id": id,
	},
})
apitest.Assert(t, resp, "status_code == 200", "$.code == 0")
```

**RPC example:**

```go
resp := apitest.CallRPC(ctx, apitest.RPCRequest{
	Method: "SearchBank",
	Body:   apitest.JSON{"Bizlines": []string{"tiktok-photo"}},
})
apitest.Assert(t, resp, "$.BaseResp.StatusCode == 0")
```

**Multi-step scenario with Extract:**

```go
// Step 1: create resource
createResp := apitest.CallHTTP(ctx, apitest.HTTPRequest{
	Method: "POST",
	Path:   "/api/v2/example/resource",
	Body:   apitest.JSON{"name": apitest.UniqueName("test")},
	Extract: map[string]string{
		"resource_id": "$.data.id",
	},
})
apitest.Assert(t, createResp, "status_code == 200", "$.code == 0")

// Step 2: query by extracted id
id := createResp.ExtractString("resource_id")
queryResp := apitest.CallHTTP(ctx, apitest.HTTPRequest{
	Method: "GET",
	Path:   "/api/v2/example/resource/detail",
	Params: map[string]string{"id": id},
})
apitest.Assert(t, queryResp, "status_code == 200", "$.data.name != ''")
```

## 4. Field-by-Field Mapping (case.md → template tokens)

| case.md / task.md / IDL field | Template token | Notes |
| --- | --- | --- |
| Interface type (HTTP path vs RPC func) | `{{REQUEST_TYPE}}` | `HTTP` when `case.md` specifies an HTTP path; `RPC` when it specifies an RPC method name. Drives `HTTPRequest`/`RPCRequest` and `CallHTTP`/`CallRPC`. |
| HTTP method + path | `Method: "<VERB>",` `Path: "<path>",` | Only for `{{REQUEST_TYPE}} == HTTP`. `<VERB>` is GET/POST/PUT/DELETE; `<path>` is the API path (e.g. `/api/v2/example/detail`). |
| RPC method name | `Method: "<RPCFuncName>",` | Only for `{{REQUEST_TYPE}} == RPC`. `<RPCFuncName>` is the PascalCase RPC function name (e.g. `SearchBank`). No `Path` or `Params` fields. |
| Method name (derived) | `{{METHOD_NAME}}`, `{{METHOD_PASCAL}}`, `{{METHOD_SNAKE}}` | Snake → file path; Pascal → Go func name |
| Go module import path | `{{APITEST_IMPORT_PATH}}` | Resolve once with `go list -m`, then append `/tests/integration/apitest`; never hardcode the current repository path. |
| Env routing (`psm/env/branch/zone/idc/cluster`) | `FEATURE_DIR/test/.env` | Generated Go must call `apitest.EnvFromFile(t)` and must not inline these fields |
| Business request fields from `case.md` / KB / schema | `Body: apitest.JSON{...}` | Prefer direct Go values; use `WithVars` only when reused by later calls |
| Each row in case.md | one `TestXxx(t)` function | Add `// TC-xxx: <测试场景>` above the function; `WithCaseID` keeps the original id |
| `Asserts` column | `apitest.Assert(t, resp, ...)` | one expression per line |
| `Extract` column | `Extract: map[string]string{...}` | varname → JSONPath. Read back via `resp.ExtractString("varname")` or `resp.ExtractInt64("varname")`. Only emit `Extract` when the value is needed by a subsequent step. |
| `### Mock Setup` block (downstream PSM + method) | Mock rule comment + optional `RpcContext` | Per `mock.md`. Emit a short Go comment near the mock-required case with `rule_name` and `mock_rule_url`. For RPC rows, also emit `RpcContext: map[string]string{"DYECP_FD_MOCK": "new_mock_tns_sdd_apitest_mock_group", "MOCK_TAG": "tns_sdd_apitest_mock_group", "APITEST_MOCK_CASE_ID": "<repo>__<caller>__<case_id>"}`. HTTP rows do not inject `RpcContext`. |

## 5. Guardrails

- **Never inline runtime env/auth fields** (`PSM/Env/Branch/Zone/IDC/Cluster`, `test_account`, cookies, Hex-Auth-Key, Authorization tokens). They live in `FEATURE_DIR/test/.env`, referenced by `APITEST_ENV`. The test must skip with a runnable hint when `APITEST_ENV` is absent.
- **Use `apitest.NewContext`** so token, log dir defaults, and gateway client setup stay centralized. Terminal users need `APITEST_ENV=<path-to-.env>` plus a paas-gw JWT from the workflow `user_jwt` step or `APITEST_TOKEN=<jwt>`; `APITEST_LOG_DIR` defaults to `api_test_logs` next to that `.env`.
- **Set `WithCaseID("TC-...")`** so logs still use the original `case.md` id.
- **Use `apitest.JSON{...}` for bodies, not raw `map[string]any`.** Keeps generated code readable.
- **Prefer ordinary Go values over placeholders.** Use `${{var}}` / `${var}` only when a value must flow through the runtime variable map.
- **One file per method.** Multiple cases become multiple `Test<Method><CaseSlug>` functions in the same package.
- **Mock traceability comments**: for mock-required cases, emit **exactly two** comment lines immediately above the mocked request block — line 1: `// Mock: <mode> (rule <rule_id>)` where `<mode>` is the `case.md` Mock Setup `Mode` column value (`data` / `panic` / `errcode_<C>` / `timeout_<N>s`); line 2: `// <mock_rule_url>` (BAM Mock console URL per `mock.md §3.6`). Do not include dyeing namespace, `Rpc-Persist-*` header, or metainfo wire-format explanations — those belong to the runtime contract, not per-case context. Do not embed mock payloads in generated Go.
- **Chain over hardcoded samples**: `resource_ref` fields (any-existing bankKey / owner / policy_group_id / ...) must be acquired by chaining a List/Query call + `resp.ExtractString(...)` from a package-level `anyExisting<X>(t, ctx)` helper that `t.Skip`s on empty environments. Do not hoist the value to a package-level `const`.
- **Dynamic construct helpers (replay-safe)**: replay-fragile fields (entity Name with uniqueness, current/past/future timestamps, random description, enum-pick) must use `apitest.UniqueName` / `apitest.NowSec` / `apitest.NowMilli` / `apitest.NowMicro` / `apitest.PastSec` / `apitest.PastMilli` / `apitest.PastMicro` / `apitest.FutureSec` / `apitest.RandString` / `apitest.RandInt` / `apitest.PickOne`. Hardcoded literals for these fields are forbidden — they break repeated runs (uniqueness collisions, stale time windows).
- **Environment business samples**: when a business field cannot be chained and its value differs across environments, declare a package-level `var envSamples = map[string]map[string]string{...}` (outer key = `env.Env`, inner key = business token) and read with `apitest.Sample(t, env, "<KEY>", envSamples)`. Missing slots `t.Skip` automatically; never fabricate a value or push it into `.env` (`.env` carries routing/auth only).

## 6. Full Example (three-section layout with chained / dynamic / env-sample helpers)

The complete shape generated by the skill when a target uses chained samples, dynamic construction, and per-env business values together:

```go
package search_bank_test

import (
	"fmt"
	"testing"

	"{{APITEST_IMPORT_PATH}}"
)

// ── ① Protocol constants (cross-env stable) ──
const (
	pathSearchBank     = "/api/v2/infa/sds/bank/search"
	mappedSDSBizline   = "tiktok_live"
	nonExistingBankKey = "__NONEXISTENT_BANK_KEY_FOR_TESTING__"
)

// ── ② Environment business samples (emit only if at least one env_business_sample field exists) ──
var envSamples = map[string]map[string]string{
	"boei18n": {},
	"prod":    {},
}

// ── ③ Test functions ──

// Chained-sample helper: acquires one existing bankKey; t.Skip on empty env.
func anyExistingBankKey(t *testing.T, ctx *apitest.TestContext) string {
	resp := apitest.CallHTTP(ctx, apitest.HTTPRequest{
		Method: "POST", Path: pathSearchBank, Body: apitest.JSON{},
	})
	key := resp.ExtractString("$.data.bankInfo[0].bankKey")
	if key == "" {
		t.Skip("no existing bank in this env; cannot drive bankKey-filter case")
	}
	return key
}

// TC-G02-02: filter by bankKeys (uses chained sample, no hardcoded literal)
func TestSearchBankByBankKeys(t *testing.T) {
	env := apitest.EnvFromFile(t)
	ctx := apitest.NewContext(t, env).WithCaseID("TC-G02-02")

	bankKey := anyExistingBankKey(t, ctx)

	resp := apitest.CallHTTP(ctx, apitest.HTTPRequest{
		Method: "POST", Path: pathSearchBank,
		Body: apitest.JSON{"bankKeys": []string{bankKey}},
	})
	apitest.Assert(t, resp,
		"status_code == 200", "$.code == 0",
		fmt.Sprintf("$.data.bankInfo[0].bankKey == '%s'", bankKey),
	)
}

// TC-...: write API uses dynamic helpers + env sample (replay-safe)
func TestCreateCustomRiskTag(t *testing.T) {
	env := apitest.EnvFromFile(t)
	ctx := apitest.NewContext(t, env).WithCaseID("TC-...")

	name := apitest.UniqueName("api_testing_crt")          // unique per run
	desc := "api testing crt " + apitest.RandString(6)     // random filler
	owner := apitest.Sample(t, env, "DEFAULT_OWNER", envSamples)

	resp := apitest.CallHTTP(ctx, apitest.HTTPRequest{
		Method: "POST", Path: "/api/v2/custom_risk_tag",
		Body: apitest.JSON{
			"name":          name,
			"description":   desc,
			"owners":        []string{owner},
			"stages":        []int{1, 2, 3},
			"releaseConfig": apitest.JSON{"isReleaseNow": true},
		},
	})
	apitest.Assert(t, resp, "$.code == 0", "$.data.customRiskTagId > 0")
}
```

