# 新移动端逐页替换现有 H5 Spec

## Why
现有 `frontend/h5` 的用户端界面需要被新的移动端界面替换，但不能做一次性大迁移，因为每个页面都涉及不同的后端接口、交互按钮和路由语义。正确路径是以新移动端页面为 Single Source of Truth，逐页完成“页面匹配、接口梳理、UI 替换、接口对接、验证记录”的闭环。

## What Changes
- 以 `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-ui/src/mobile` 作为新移动端 UI 来源。
- 以 `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5` 作为被替换的 H5 工程。
- 建立新旧页面映射，优先处理新移动端存在的页面。
- 每个页面独立迁移：先找新页面，再找旧页面，再梳理旧页面接口，再替换 UI，再对接接口和按钮功能。
- 新移动端有、旧 H5 没有的页面 SHALL 保留并接入到 H5 路由。
- 旧 H5 有、新移动端没有的页面 SHALL 从用户可达路径中舍弃，但不得在未经确认前删除旧代码。
- 旧接口冗余时 SHALL 记录到待确认文档，不直接删除后端接口或服务封装。
- 新 UI 需要但现有后端缺失的接口 SHALL 记录到单独文档，用于后续后端补齐。
- 迁移完成后 SHALL 明确询问用户是否删除旧移动端代码。

## Impact
- Affected specs: H5 用户端页面、H5 路由、H5 接口调用、移动端 UI 迁移、接口差异记录。
- Affected code: `frontend/h5/src/App.tsx`、`frontend/h5/src/pages/**`、`frontend/h5/src/components/**`、`frontend/h5/src/services/**`、`frontend/h5/src/store/**`、`frontend/h5/src/hooks/**`、`frontend/h5/src/styles/**`、`frontend/h5/e2e/**`、`frontend/h5/src/__tests__/**`。
- Source UI reference: `dy-ai-live-auction-fullstack-ui/src/mobile/App.tsx`、`dy-ai-live-auction-fullstack-ui/src/mobile/pages/**`、`dy-ai-live-auction-fullstack-ui/src/mobile/components/**`、`dy-ai-live-auction-fullstack-ui/src/mobile/index.css`。
- Documentation deliverables during implementation: `frontend/h5/docs/mobile-ui-migration/page-mapping.md`、`frontend/h5/docs/mobile-ui-migration/redundant-interfaces.md`、`frontend/h5/docs/mobile-ui-migration/missing-interfaces.md`、`frontend/h5/docs/mobile-ui-migration/migration-progress.md`。

## ADDED Requirements

### Requirement: Page Mapping Inventory
The system SHALL maintain a migration inventory that maps every new mobile page to its old H5 counterpart, route, existing API dependencies, target behavior, and migration status.

#### Scenario: New page has old counterpart
- **WHEN** the new mobile page exists and an old H5 page provides the same business capability
- **THEN** the inventory records both paths, the old route, the new target route, and the interfaces used by the old page

#### Scenario: New page has no old counterpart
- **WHEN** the new mobile page has no matching old H5 page
- **THEN** the inventory records it as a new retained page and includes the route that will make it reachable in H5

#### Scenario: Old page has no new counterpart
- **WHEN** an old H5 page has no matching new mobile page
- **THEN** the inventory records it as discarded from navigation while preserving source files until user confirms deletion

### Requirement: Page-By-Page Migration
The system SHALL migrate one page at a time and complete interface integration for that page before starting the next page.

#### Scenario: Migrate one mapped page
- **WHEN** a page is selected for migration
- **THEN** implementation first identifies the new UI page, then identifies the old H5 page, then lists old API calls, then replaces the old UI, then wires required APIs and buttons

#### Scenario: Current page is not fully integrated
- **WHEN** UI replacement is done but API or button behavior is incomplete
- **THEN** the next page migration SHALL NOT start

### Requirement: New UI Is Source Of Truth
The system SHALL prefer the new mobile UI feature set when old H5 features or interfaces conflict with the new page.

#### Scenario: Old page has extra interface
- **WHEN** the old H5 page calls an interface for a feature absent from the new mobile page
- **THEN** the interface is treated as redundant for the page and recorded in `redundant-interfaces.md`

#### Scenario: New page needs missing backend capability
- **WHEN** the new mobile page requires data or actions not supported by existing H5 service wrappers or backend endpoints
- **THEN** the missing capability is recorded in `missing-interfaces.md` with page, action, expected request, expected response, and current gap

#### Scenario: Interface response mismatches new UI
- **WHEN** an existing interface response does not match the data shape required by the new UI
- **THEN** the frontend SHALL adapt mapping types first if possible, and record backend contract mismatches when frontend adaptation would hide missing domain data

### Requirement: Route And Navigation Replacement
The system SHALL update H5 routing and navigation so users land on the migrated new mobile pages.

#### Scenario: User opens migrated route
- **WHEN** the user opens a route that maps to a migrated page
- **THEN** the H5 app renders the new mobile page implementation and preserves expected authentication behavior

#### Scenario: User navigates through bottom navigation
- **WHEN** the new mobile UI provides bottom navigation
- **THEN** navigation targets SHALL align with H5 routes and only expose retained pages

### Requirement: Interface Documentation
The system SHALL produce migration documentation for redundant interfaces and missing backend interfaces before completion.

#### Scenario: Redundant old interface exists
- **WHEN** an old interface is no longer required by the new UI
- **THEN** it is documented for final confirmation and not silently removed from backend or shared services

#### Scenario: Missing new interface exists
- **WHEN** a new UI function cannot be fully connected due to missing backend support
- **THEN** it is documented as a backend gap and the frontend behavior uses an explicit safe fallback

### Requirement: Completion Prompt
The system SHALL ask whether to delete old mobile code only after all page migrations, interface integrations, and verification checkpoints are complete.

#### Scenario: Migration is complete
- **WHEN** all checklist items pass
- **THEN** the final response states the migration is complete and asks whether the user wants to delete old mobile code

## MODIFIED Requirements

### Requirement: H5 User Interface
The H5 user-facing mobile interface SHALL use the new mobile UI pages from `dy-ai-live-auction-fullstack-ui/src/mobile` as the target presentation and interaction model, while retaining the existing H5 project runtime, gateway routing, auth context, error handling, and backend integration conventions where still applicable.

### Requirement: H5 API Integration
The H5 API integration SHALL only connect interfaces needed by retained new mobile UI features. Existing API wrappers may remain for compatibility, but page-level usage SHALL be removed when the new page no longer has the corresponding feature.

### Requirement: H5 Navigation Scope
The H5 navigation SHALL expose pages present in the new mobile UI target set: `Home`、`LiveRoom`、`ProductDetail`、`AuctionResult`、`Profile`、`AuctionHistory`、`Notifications`、`Following`、`Login`，unless implementation discovers a page is purely decorative or unreachable in the new source app.

## REMOVED Requirements

### Requirement: Preserve Old H5 Pages As User-Reachable UI
**Reason**: The replacement target is the new mobile interface; old H5-only pages conflict with the new UI as SSOT.
**Migration**: Remove old H5-only pages from active navigation and routes when no new counterpart exists. Keep source files until the final user confirmation about deletion.
