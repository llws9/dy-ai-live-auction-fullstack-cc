# apitest — Go Runtime for API Integration Tests

`apitest` is the only runtime this skill targets. Runtime details live in `tests/integration/apitest/README.md`.

Library import path: resolve per repository with `go list -m`, then append `/tests/integration/apitest`. Do not hardcode this skill's home repository path into generated tests.

## 1. Skill responsibilities

When the api-test skill runs, it must:

1. **Generate** one `*_test.go` per target, following `resources/go_test_template.md`.
2. **Construct** request bodies per `go_driver/SKILL.md ## Decisions: fill request field`. `case.md` overrides always win; IDL shape resolved per `## Decisions: where to get IDL shape`.
3. **Wire** assertions / extracts from `case.md` into `apitest.Assert(...)` and request `Extract` maps.
4. **Use** `apitest.NewContext(...)`. It centralizes `APITEST_TOKEN`, `APITEST_LOG_DIR`, and `APITEST_ENV` handling so generated tests stay flat and readable.
5. **Skip secrets**: `test_account` headers (cookie / Hex-Auth-Key / Authorization) are loaded at runtime via `os.Getenv("APITEST_ENV")`, never inlined.

## 2. Wire format

| Concern | Behavior |
| --- | --- |
| Gateway endpoints | Selected inside `gateway.go` from `.env zone` / `idc` / `env` using the Explorer OpenAPI control-plane table: CN `paas-gw.byted.org`, BOE `paas-gw-boe.byted.org`, BOEI18N/BOETTP `paas-gw-boei18n.byted.org`, I18N office `bc-useastdt-gw.tiktok-row.net`, GCP `paas-gw-gcp.tiktoke.org`, TTP `paas-gw-tx.tiktokd.org`, SINF `paas-gw.sinf.net`. Do not add gateway domain fields to `.env`. |
| Outer headers | `X-Jwt-Token` from the workflow `user_jwt` step or `$APITEST_TOKEN`, `Domain: explorer`, `Content-Type: application/json` |
| Inner payload | JSON envelope with `psm/host/zone/idc/cluster/env/path/method/header/request/func_name/...` (see `gateway.go`) |
| Variable syntax | `${{var}}` preserves type (use inside JSON literals); `${var}` always stringifies (use inside URL paths / strings) |
| Assertion grammar | `status_code == 200`, `$.x == y`, `typeof $.x == 'int'`, `len($.x) > 0`, `jsonpath('$.x') in [1, 2, 3]`, `jsonpath('$.x') == y` — powered by the built-in assertion engine in `assert.go` |
| Log filename | `<logDir>/apitest_<case_id>.log`, sections compatible with `resources/test_report_guide.md` |
| PASS conditions | gateway HTTP 200 AND `has_permission == true` AND every assertion truthy |

## 3. Skill-side execution contract

After the skill writes a target test package, it runs:

```bash
export APITEST_TOKEN=<paas-gw JWT>
export APITEST_ENV=<absolute path to FEATURE_DIR/test/.env>
export APITEST_LOG_DIR=<absolute path to FEATURE_DIR/test/api_test_logs>
GOWORK=off go test -v -count=1 -run Test<MethodPascal> ./tests/integration/<method_snake>/...
```

For terminal-only runs, provide `APITEST_ENV` and export `APITEST_TOKEN` with a paas-gw JWT. `APITEST_LOG_DIR` is optional and defaults to `api_test_logs` next to `APITEST_ENV`.

Flags: `-count=1` (no cache) / `-run` (scoped) / `GOWORK=off` (rgo-cached deps may be absent locally — drop if your workspace is complete). Generated tests self-skip with a runnable hint when `APITEST_ENV` or paas-gw JWT is unresolved. Log location: see §1.4.

## 4. Guardrails

- The runtime is the Go library + `go test`, nothing else. Do not introduce shell-outs to any external CLI.
- The single test-suite format is ordinary `*_test.go` code using `apitest.NewContext`, `HTTPRequest` / `RPCRequest`, `CallHTTP` / `CallRPC`, and `Assert`. Do not introduce a parallel YAML / DSL representation of cases.
- Cookies / tokens / Hex-Auth-Key / Hex-Login-User-Info MUST stay in the developer-local `.env` referenced by `os.Getenv("APITEST_ENV")`; never inline them into the committed `*_test.go`.

## 5. Runtime maintenance & versioning

The `apitest` runtime is shipped two ways simultaneously:

| Where it lives | Role |
|---|---|
| Per-repo `tests/integration/apitest/` | The actual Go package every generated test imports. Vendored, tracked in git, owned by the repo. |
| Skill-internal `go_driver/runtime/` (vendored baseline) + `runtime/manifest.json` | The skill's source-of-truth baseline. Drives `SKILL.md` §1.1 Generate 第 1 步（运行时同步 — scaffold / upgrade / drift detection — runs once per skill invocation, before feature-level work). |

### 5.1 Versioning contract (semver, enforced by Workflow §0)

Both `runtime/version.go` and the per-repo `tests/integration/apitest/version.go` expose `const Version = "X.Y.Z"`. The number is a **stability contract** that constrains how a runtime change must look:

| Bump kind | Allowed | Forbidden | Workflow §0 behavior |
|---|---|---|---|
| `patch` (X.Y.Z+1) | bug fix; comment / log / impl-internal change | **any** public symbol added, removed, renamed, or signature-changed | silent overwrite after compile-verify |
| `minor` (X.Y+1.0) | adding new public symbols; loosening a parameter; new optional behavior | renaming / removing existing public symbols, breaking JSON shape | overwrite after compile-verify; if compile fails, rollback + ask user |
| `major` (X+1.0.0) | anything | none — but you MUST populate `manifest.breaking_changes[]` and bump `Version` accordingly | overwrite after compile-verify; if compile fails, rollback + ask user (same as minor; users are expected to read `breaking_changes` first) |

### 5.2 How to bump the runtime version

Treat this as a **single atomic change set** — do all five steps in one MR:

1. **Edit the source** in `go_driver/runtime/*.go` first (NOT the per-repo `tests/integration/apitest/`; the latter gets pushed by `SKILL.md` §1.1 Generate 第 1 步 on next skill run).
2. **Bump `runtime/version.go`** `Version` per §5.1. Patch / minor / major decide here.
3. **Recompute sha256** for every changed file and update `runtime/manifest.json` `files[].sha256`. Also bump `manifest.json` `version` to match `version.go`.
4. **For minor / major**: append an entry to `manifest.breaking_changes[]` with `from`, `to`, a one-line `summary`, and a `migration_hint` (concrete sed / regex / patch suggestion).
5. **Hand-test once** in this repo (the skill's home repo): run any generation flow that exercises the changed code path; confirm `tests/integration/apitest/` gets upgraded smoothly, generated imports use `<go list -m>/tests/integration/apitest`, and the existing tests still compile.

`source_commit` in `manifest.json` is informational — bump it whenever you re-import from an upstream source so the trail stays auditable.

The skill assumes the runtime lives at `<go module>/tests/integration/apitest` (vendored per repo, kept in sync by Workflow §0).
