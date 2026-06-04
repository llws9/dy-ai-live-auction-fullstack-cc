# Deploy Ignore Gitignored Changes Design

## 目标

调整 `/dp-dev` 和 `/dp-prod` 的部署安全检查：当本地存在未提交改动时，如果这些改动的路径命中 `.gitignore` 规则，则部署可以继续；脚本不得删除、还原、stash 或覆盖这些改动。

核心目标是区分两类本地状态：

- 真实源码/配置改动：继续阻断部署。
- 本地构建产物、缓存或其他被 `.gitignore` 覆盖的改动：允许忽略并继续部署。

## 背景

当前 `scripts/deploy-dev.sh` 和 `scripts/deploy-prod.sh` 都使用 `git status --porcelain` 判断工作区是否干净。

这个判断过于粗糙。以 `frontend/admin/dist/index.html` 为例：

- `.gitignore` 中有 `dist/`。
- `git check-ignore --no-index -v frontend/admin/dist/index.html` 能确认它命中 `.gitignore`。
- 但该文件已被 Git 跟踪，所以 `git status --porcelain` 仍显示 `M frontend/admin/dist/index.html`。

用户期望这类命中 `.gitignore` 的本地改动不阻断部署。

## 适用范围

规则同时适用于：

- `/dp-dev`，对应 `scripts/deploy-dev.sh restart`
- `/dp-prod`，对应 `scripts/deploy-prod.sh apply`

不改变以下原则：

- 部署代码来源仍必须是 `origin/main`。
- `HEAD` 仍必须等于 `origin/main`。
- 不命中 `.gitignore` 的本地改动仍必须阻断部署。
- 脚本不得自动丢弃用户本地改动。

## 判定规则

新增工作区检查函数，替代当前的完全干净检查。

函数逻辑：

1. 执行 `git status --porcelain` 获取本地改动列表。
2. 如果没有改动，检查通过。
3. 对每条改动解析出路径。
4. 对路径执行 `git check-ignore --no-index -q -- <path>`。
5. 如果路径命中 `.gitignore`，加入 ignored-local changes 列表。
6. 如果路径不命中 `.gitignore`，加入 blocking changes 列表。
7. 如果 blocking changes 非空，停止部署并输出阻断项。
8. 如果只有 ignored-local changes，输出提示并继续部署。

`--no-index` 是必要的，因为 Git 默认不会对已跟踪文件应用 ignore 规则；本需求要求即使文件已跟踪，只要路径命中 `.gitignore`，也允许忽略。

## 路径解析

`git status --porcelain` 可能输出：

- ` M path`
- `M  path`
- `?? path`
- `A  path`
- `R  old -> new`
- `C  old -> new`

脚本只需要覆盖部署风险相关路径提取：

- 普通改动取状态字段之后的路径。
- rename/copy 取 `->` 右侧的新路径。
- 路径传给 `git check-ignore` 时必须使用 `--` 防止特殊路径被误解析为参数。

## 输出行为

如果只存在可忽略改动，输出：

```text
检测到仅含 .gitignore 覆盖的本地改动，部署将忽略这些文件：
- <path>
```

如果存在阻断改动，输出：

```text
错误: 当前工作区存在未提交且未被 .gitignore 覆盖的改动，拒绝部署
- <path>
```

## 修改文件

- `scripts/deploy-dev.sh`
  - 将 `assert_clean_tree` 改为“阻断非 ignored 改动”的检查。
- `scripts/deploy-prod.sh`
  - 将 `assert_clean_for_ref` 中的工作区检查改为同样规则。
- `.trae/skills/project-deploy/SKILL.md`
  - 更新 `/dp-dev` 和 `/dp-prod` 的安全规则说明。
- `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`
  - 同步全局 skill 说明，避免本地与全局行为描述不一致。

## 验收标准

### 仅 ignored 改动

当前存在 `frontend/admin/dist/index.html` 修改，且它命中 `.gitignore` 时：

- `scripts/deploy-dev.sh restart` 不因该文件阻断。
- `scripts/deploy-prod.sh apply` 不因该文件阻断。
- 脚本输出 ignored-local changes 提示。

### 非 ignored 改动

如果修改 `README.md` 等不命中 `.gitignore` 的文件：

- `scripts/deploy-dev.sh restart` 必须阻断。
- `scripts/deploy-prod.sh apply` 必须阻断。
- 输出阻断文件路径。

### Git 指针不一致

如果 `HEAD != origin/main`：

- 部署仍必须阻断。
- 不能因为本地改动均命中 `.gitignore` 而绕过提交指针检查。

## 非目标

- 不自动修改 `.gitignore`。
- 不自动 `git rm --cached` 已跟踪构建产物。
- 不自动 stash、checkout、reset 或删除任何用户改动。
- 不改变线上部署确认门禁。
