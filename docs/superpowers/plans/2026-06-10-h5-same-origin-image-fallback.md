# H5 Same-Origin Image Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove production H5 dependency on `copilot-cn.bytedance.net` image URLs by using a same-origin static fallback image and repairing demo fixture defaults.

**Architecture:** H5 owns a local SVG fallback under `public/`, so Vite/Nginx serves it from the same origin as the app. Frontend fallback constants, H5 E2E fixtures, and the E2E test SDK default fixture image point to the same public path. Production data repair is handled as a separate verified SQL update against existing demo records that still contain the internal image domain.

**Tech Stack:** React, Vite public assets, Jest, Go test, MySQL JSON update via demo deployment shell.

---

### Task 1: Lock H5 Homepage Fallback Behavior

**Files:**
- Modify: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx`

- [ ] **Step 1: Write the failing test expectation**

Change the existing fallback assertion from the internal generated image API to a same-origin asset path:

```ts
const image = await screen.findByRole('img', { name: '压测拍品 1780733852' });
expect(image).toHaveAttribute('src', '/assets/default-auction-cover.svg');
expect(image).not.toHaveAttribute('src', expect.stringContaining('copilot-cn.bytedance.net'));
expect(screen.queryByText('暂无图片')).not.toBeInTheDocument();
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5
npm test -- --runInBand src/pages/Home/__tests__/Home.test.tsx -t '首页竞拍卡片在商品无图时使用兜底图片'
```

Expected: FAIL because current `DEFAULT_PRODUCT_COVER_IMAGE` still points to `copilot-cn.bytedance.net`.

### Task 2: Add Same-Origin H5 Static Asset

**Files:**
- Create: `frontend/h5/public/assets/default-auction-cover.svg`
- Modify: `frontend/h5/src/pages/Home/index.tsx`
- Modify: `frontend/h5/src/utils/imageFallback.ts`
- Modify: `frontend/h5/src/components/FixedPriceCard/__tests__/FixedPriceCard.test.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/BidDock.test.tsx`
- Modify: `frontend/h5/e2e/utils/new-ui-fixtures.ts`
- Modify: `frontend/h5/e2e/fixed-price.spec.ts`

- [ ] **Step 1: Add SVG fallback asset**

Create `frontend/h5/public/assets/default-auction-cover.svg`:

```svg
<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480" viewBox="0 0 640 480" role="img" aria-labelledby="title desc">
  <title id="title">Default auction product cover</title>
  <desc id="desc">Warm premium auction product placeholder with abstract jewelry and watch shapes.</desc>
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#fff8ec"/>
      <stop offset="0.5" stop-color="#f4e3c2"/>
      <stop offset="1" stop-color="#241c17"/>
    </linearGradient>
    <radialGradient id="glow" cx="38%" cy="30%" r="60%">
      <stop offset="0" stop-color="#ffffff" stop-opacity="0.78"/>
      <stop offset="1" stop-color="#ffffff" stop-opacity="0"/>
    </radialGradient>
  </defs>
  <rect width="640" height="480" fill="url(#bg)"/>
  <rect width="640" height="480" fill="url(#glow)"/>
  <circle cx="450" cy="174" r="82" fill="#1d2433" opacity="0.9"/>
  <circle cx="450" cy="174" r="55" fill="none" stroke="#c9a96e" stroke-width="12"/>
  <circle cx="230" cy="252" r="68" fill="#2f7d6d" opacity="0.92"/>
  <path d="M184 252c22-42 69-60 114-42-17 40-64 76-114 42Z" fill="#bce7d7" opacity="0.85"/>
  <path d="M116 357h408" stroke="#7b5b2d" stroke-width="18" stroke-linecap="round" opacity="0.38"/>
  <text x="320" y="410" text-anchor="middle" font-family="Inter, PingFang SC, Arial, sans-serif" font-size="34" font-weight="700" fill="#6f4f20">AUCTION</text>
</svg>
```

- [ ] **Step 2: Point H5 default image to same-origin path**

Update `frontend/h5/src/pages/Home/index.tsx`:

```ts
const DEFAULT_PRODUCT_COVER_IMAGE = '/assets/default-auction-cover.svg';
```

Update `frontend/h5/src/utils/imageFallback.ts`:

```ts
export const DEFAULT_AUCTION_IMAGE = '/assets/default-auction-cover.svg';
```

Update H5 E2E fixtures so mocked products also use `/assets/default-auction-cover.svg`, preventing test-created data from reintroducing the internal domain.

- [ ] **Step 3: Run H5 targeted test**

Run:

```bash
cd frontend/h5
npm test -- --runInBand src/pages/Home/__tests__/Home.test.tsx -t '首页竞拍卡片在商品无图时使用兜底图片'
```

Expected: PASS.

### Task 3: Lock Test SDK Default Product Image

**Files:**
- Modify: `backend/test/client/auction/client_test.go`
- Modify: `backend/test/client/auction/client.go`

- [ ] **Step 1: Write failing SDK test expectation**

Change `TestSDK_CreateProductAsAddsDefaultImageWhenMissing`:

```go
if captured.Images[0] != "/assets/default-auction-cover.svg" {
	t.Fatalf("default image should use same-origin H5 asset, got %q", captured.Images[0])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend/test
go test ./client/auction -run TestSDK_CreateProductAsAddsDefaultImageWhenMissing -count=1
```

Expected: FAIL because current SDK default image is `copilot-cn.bytedance.net`.

- [ ] **Step 3: Update SDK default**

Update `backend/test/client/auction/client.go`:

```go
defaultFixtureProductImage = "/assets/default-auction-cover.svg"
```

- [ ] **Step 4: Run SDK targeted test**

Run:

```bash
cd backend/test
go test ./client/auction -run TestSDK_CreateProductAsAddsDefaultImageWhenMissing -count=1
```

Expected: PASS.

### Task 4: Regression Scan And Build Check

**Files:**
- Read-only scan across repository.

- [ ] **Step 1: Scan for remaining internal image defaults**

Run:

```bash
Use the repository search tool to scan for:

```text
copilot-cn.bytedance.net/api/ide/v1/text_to_image
```

Targets:

```text
frontend/h5
backend/test/client/auction
```
```

Expected: no matches in the H5 runtime fallback or auction SDK default.

- [ ] **Step 2: Run focused H5 build**

Run:

```bash
cd frontend/h5
npm run build
```

Expected: PASS and `dist/assets/default-auction-cover.svg` exists.

### Task 5: Production Existing Data Repair

**Files:**
- No source code changes unless deployment scripts already have a safe seed/repair hook.

- [ ] **Step 1: Verify old data still contains internal image domain**

Run against demo API:

```bash
python3 - <<'PY'
import json, urllib.request
url = 'http://14.103.53.55/api/v1/auctions?page=1&page_size=20'
with urllib.request.urlopen(url, timeout=10) as r:
    data = json.load(r)
for item in data.get('data', {}).get('list', []):
    image = item.get('product', {}).get('image', '')
    if 'copilot-cn.bytedance.net' in image:
        print(item['product_id'], image)
PY
```

Expected before repair: one or more product IDs print.

- [ ] **Step 2: Repair product image JSON in demo MySQL**

Run on the server only after code is built and synced:

```bash
ssh -i /Users/bytedance/Downloads/dy-auction.pem root@14.103.53.55 '
  cd /srv/auction/app &&
  docker compose --project-name auction-demo --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml exec -T mysql \
    mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" \
    -e "UPDATE products SET images = JSON_ARRAY('\''/assets/default-auction-cover.svg'\'') WHERE JSON_SEARCH(images, '\''one'\'', '\''%copilot-cn.bytedance.net%'\'') IS NOT NULL;"
'
```

Expected: command exits `0`; no secrets are printed.

- [ ] **Step 3: Verify API no longer returns internal image domain**

Run the same Python script from Step 1.

Expected after repair: no product IDs print.

### Task 6: Deploy H5 Static Fix

**Files:**
- Use existing deployment script SSOT.

- [ ] **Step 1: Generate production plan**

Run:

```bash
scripts/deploy-prod.sh plan
```

Expected: plan shows H5 frontend changes and target commit.

- [ ] **Step 2: Ask for explicit production confirmation**

Required prompt:

```text
确认执行线上部署吗？回复“确认部署”后我才会执行 apply。
```

- [ ] **Step 3: Apply after confirmation only**

Run:

```bash
scripts/deploy-prod.sh apply
scripts/deploy-prod.sh verify
```

Expected: both commands exit `0`; H5 root and `/api/v1/auctions` verify.

### Self-Review

- Spec coverage: Covers H5 runtime fallback, SDK default fixture data, existing production data repair, and deployment verification.
- Placeholder scan: No TBD/TODO placeholders.
- Type consistency: H5 fallback is a string path; SDK default remains `[]string`; SQL updates JSON array content to the same path.
