# Environment Configuration Template

This document provides the standard environment configuration template for services running in the PPE (Pre-Production Environment) or BOE (Bytedance Offline Environment). 

## Field Definitions

The following fields are used to configure a target service environment. Ensure all required fields are populated correctly.

| Field | Type | Required | Description | Default |
| --- | --- | --- | --- | --- |
| `psm` | String | Yes | The unique identifier for the service. The format is typically `tiktok.demo.<service_name>`. If not provided, please ask the user to supply it. | |
| `host` | String | No | The hostname for the service endpoint. This value is sourced from the `Host` field in the service's `FEATURE_DIR/test/task.md` file. | |
| `env` | String | Yes | The name of the PPE/BOE environment where the service is deployed. This is sourced from the `Env` field in `FEATURE_DIR/test/task.md`. | |
| `branch` | String | Yes | The IDL branch. This is sourced from the `IDL_Branch`(preferred) or `Branch` field in `FEATURE_DIR/test/task.md`. | |
| `zone` | String | Yes | The mapped geographical region code. This value is derived from the `VRegion` field in `FEATURE_DIR/test/task.md` and mapped using the `zone_mapping.md` reference. | |
| `idc` | String | No | The specific IDC (Internet Data Center) where the service is running. This is sourced from the `IDC` field in `FEATURE_DIR/test/task.md`. | |
| `cluster`| String | No | The cluster name within the IDC. This is sourced from the `Cluster` field in `FEATURE_DIR/test/task.md`. | `default` |
| `test_account`| Object | Required for non-public HTTP APIs unless `cookie` is provided | Contains authentication headers required to access the service. Populate from user-provided test account data for this feature; do not derive from sibling tests or business KB. For authenticated HTTP endpoints, either this object or `cookie` must be provided before execution. | `{}` |
| `cookie` | String | Required for non-public HTTP APIs unless `test_account` is provided | Session cookie for authentication. Part of the `test_account` object when nested there, but a cookie value alone is also an acceptable HTTP auth input when the API uses cookie-based login. | ` ` |


## Configuration Template

Use the following structure as the `FEATURE_DIR/test/.env` (`APITEST_ENV`) configuration template. The placeholders `[PSM]`, `[HOST]`, etc., should be replaced with actual values based on the field definitions above.

```yaml
# tiktok.demo.psm1
- psm: [PSM]          # format: tiktok.demo.psm1. Ask user to provide it if not provided.
  host: [HOST]        # Source: `Host` in `FEATURE_DIR/test/task.md`
  env: [ENV]          # Source: `PPE Environment Name` in `FEATURE_DIR/test/task.md`
  branch: [BRANCH]    # Source: `Branch` in `FEATURE_DIR/test/task.md`
  zone: [ZONE]        # Source: `VRegion` in `FEATURE_DIR/test/task.md`, map using `zone_mapping.md`
  idc: [IDC]          # Source: `IDC` in `FEATURE_DIR/test/task.md`
  cluster: [CLUSTER]  # Source: `Cluster` in `FEATURE_DIR/test/task.md`. Defaults to "default"
  test_account:       # Default value is empty. Fill only from user-provided feature test account data.
    cookie:           # Default value is empty.
    # Other authorization headers can be added here, e.g., JWT tokens.
    # Authorization: Bearer [TOKEN]

```

## Worked Examples

Here are two realistic examples with dummy data.

**Example 1: `tiktok.demo.psm1`**

```yaml
- psm: tiktok.demo.psm1
  host: ppe.demo-service1.tiktok.com
  env: ppe-music-service-us
  branch: feature/new-recommendation-flow
  zone: us-east-1
  idc: va6
  cluster: default
  test_account:
    cookie: "sessionid=abc123xyz456; user_id=987654321; other_flag=true"
    Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkRlbW8gVXNlciIsImlhdCI6MTUxNjIzOTAyMn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

**Example 2: `tiktok.demo.psm2`**

```yaml
- psm: tiktok.demo.psm2
  host: ppe.demo-service2.tiktok.com
  env: ppe-live-streaming-eu
  branch: hotfix/payment-gateway-bug
  zone: eu-central-1
  idc: fr5
  cluster: live-cluster-a
  test_account:
    cookie: "sessionid=def456uvw789; user_id=123456789; region=de"
    custom_auth_header: "custom token"
```

## Usage Notes

*   **Sourcing Values**: Env routing values come from `FEATURE_DIR/test/task.md`; auth/test-account values come from user-provided feature test account data. For non-public HTTP targets, collect either `test_account` headers or a `cookie` value before execution; if neither is available, ask the user which auth method to use or stop/skip with a clear reason.
*   **Gateway Selection**: runtime infers the paas-gw control plane from `zone` / `idc` / `env` inside the gateway layer. Do not add gateway domain fields to `.env`. The built-in mapping follows the Explorer OpenAPI domain table: CN → `https://paas-gw.byted.org/api/v1`; BOE → `https://paas-gw-boe.byted.org/api/v1`; BOEI18N/BOETTP → `https://paas-gw-boei18n.byted.org/api/v1`; I18N office → `https://bc-useastdt-gw.tiktok-row.net/api/v1`; GCP → `https://paas-gw-gcp.tiktoke.org/api/v1`; TTP/TTP2 → `https://paas-gw-tx.tiktokd.org/api/v1`; SINF BOE → `https://paas-gw-boe.sinf.net/api/v1`; SINF online → `https://paas-gw.sinf.net/api/v1`.
*   **Defaults**: If the `cluster` field is not specified in `task.md`, you can safely use the default value `default`.
*   **Terminal runs**: Generated Go tests read routing and business auth headers from `APITEST_ENV=<path-to-this-file>`. Paas-gw JWT is not stored in `.env`; get it through `user_jwt` during the workflow or export `APITEST_TOKEN=<jwt>` before running tests manually. `APITEST_LOG_DIR` is optional and defaults to `api_test_logs` next to this file.
```