# Deploy Ignore Gitignored Changes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow `/dp-dev` and `/dp-prod` deployment checks to ignore local changes whose paths match `.gitignore`, including already tracked files.

**Architecture:** Keep the existing `HEAD == origin/main` safety check unchanged. Replace the all-or-nothing `git status --porcelain` clean-tree check with a helper that classifies changes into ignored-local changes and blocking changes by using `git check-ignore --no-index`.

**Tech Stack:** Bash, Git CLI, Trae project skill markdown.

---

## File Structure

- Modify: `scripts/deploy-dev.sh`
  - Replace `assert_clean_tree` with a check that blocks only non-ignored local changes.
- Modify: `scripts/deploy-prod.sh`
  - Replace the worktree-clean portion of `assert_clean_for_ref` with the same ignored-change rule.
- Modify: `.trae/skills/project-deploy/SKILL.md`
  - Document that `.gitignore`-matched local changes do not block `/dp-dev` or `/dp-prod`.
- Modify: `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`
  - Keep global skill text consistent with the project skill.

Do not stage or commit `frontend/admin/dist/index.html`; it is an existing local ignored-path change used for verification.

---

### Task 1: Update Deployment Scripts

**Files:**
- Modify: `scripts/deploy-dev.sh`
- Modify: `scripts/deploy-prod.sh`

- [ ] **Step 1: Add shared helper functions to `scripts/deploy-dev.sh`**

Insert these functions immediately after `assert_origin_main` and before `assert_clean_tree`:

```bash
status_path() {
  local line=$1
  local path="${line:3}"
  if [[ "$path" == *" -> "* ]]; then
    path="${path##* -> }"
  fi
  echo "$path"
}

classify_worktree_changes() {
  local status
  status="$(git status --porcelain)"
  if [[ -z "$status" ]]; then
    return 0
  fi

  local ignored=()
  local blocking=()
  local line
  local path

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    path="$(status_path "$line")"
    if git check-ignore --no-index -q -- "$path"; then
      ignored+=("$path")
    else
      blocking+=("$path")
    fi
  done <<< "$status"

  if [[ "${#ignored[@]}" -gt 0 ]]; then
    echo -e "${YELLOW}检测到仅含 .gitignore 覆盖的本地改动，部署将忽略这些文件：${NC}"
    printf -- '- %s\n' "${ignored[@]}"
  fi

  if [[ "${#blocking[@]}" -gt 0 ]]; then
    echo -e "${RED}错误: 当前工作区存在未提交且未被 .gitignore 覆盖的改动，拒绝部署${NC}" >&2
    printf -- '- %s\n' "${blocking[@]}" >&2
    exit 1
  fi
}
```

- [ ] **Step 2: Replace `assert_clean_tree` in `scripts/deploy-dev.sh`**

Replace the full function:

```bash
assert_clean_tree() {
  cd "$PROJECT_ROOT"
  if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}错误: 当前工作区存在未提交改动，拒绝从本目录强制部署${NC}" >&2
    git status --short
    echo "建议先提交/暂存改动，或创建干净 worktree 后执行 /dp-dev。"
    exit 1
  fi
}
```

with:

```bash
assert_clean_tree() {
  cd "$PROJECT_ROOT"
  classify_worktree_changes
}
```

- [ ] **Step 3: Add the same helper functions to `scripts/deploy-prod.sh`**

Insert the same `status_path` and `classify_worktree_changes` functions immediately after `remote_sha` and before `assert_clean_for_ref`.

- [ ] **Step 4: Replace clean-tree logic in `scripts/deploy-prod.sh`**

Inside `assert_clean_for_ref`, replace:

```bash
  if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}错误: 当前工作区存在未提交改动，拒绝线上部署${NC}" >&2
    git status --short
    exit 1
  fi
```

with:

```bash
  classify_worktree_changes
```

- [ ] **Step 5: Validate script syntax**

Run:

```bash
bash -n scripts/deploy-dev.sh
bash -n scripts/deploy-prod.sh
```

Expected: both commands exit with code `0`.

- [ ] **Step 6: Verify ignored tracked change is allowed**

Run:

```bash
git check-ignore --no-index -v frontend/admin/dist/index.html
scripts/deploy-dev.sh restart
```

Expected:

```text
检测到仅含 .gitignore 覆盖的本地改动，部署将忽略这些文件：
- frontend/admin/dist/index.html
```

Expected: `scripts/deploy-dev.sh restart` does not fail because of `frontend/admin/dist/index.html`.

- [ ] **Step 7: Verify non-ignored change is blocked**

Create a temporary non-ignored change:

```bash
printf '\n<!-- deploy-check-test -->\n' >> README.md
scripts/deploy-dev.sh restart
```

Expected: command exits non-zero and prints:

```text
错误: 当前工作区存在未提交且未被 .gitignore 覆盖的改动，拒绝部署
- README.md
```

Restore only the test edit:

```bash
python3 - <<'PY'
from pathlib import Path
p = Path('README.md')
text = p.read_text()
text = text.replace('\n<!-- deploy-check-test -->\n', '')
p.write_text(text)
PY
```

Expected: `git diff -- README.md` prints no diff.

- [ ] **Step 8: Verify prod apply precheck path without remote mutation**

Do not run `scripts/deploy-prod.sh apply` because it performs online deployment. Instead verify the helper is present and syntax-safe:

```bash
grep -n "classify_worktree_changes" scripts/deploy-prod.sh
bash -n scripts/deploy-prod.sh
```

Expected: `grep` shows the helper definition and call inside `assert_clean_for_ref`.

- [ ] **Step 9: Commit script changes**

Stage only the scripts:

```bash
git add scripts/deploy-dev.sh scripts/deploy-prod.sh
git diff --cached --name-only
git commit -m "fix: ignore gitignored deploy changes"
```

Expected staged files:

```text
scripts/deploy-dev.sh
scripts/deploy-prod.sh
```

---

### Task 2: Update Skill Documentation

**Files:**
- Modify: `.trae/skills/project-deploy/SKILL.md`
- Modify outside repo: `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`

- [ ] **Step 1: Update project skill safety rules**

In `.trae/skills/project-deploy/SKILL.md`, replace:

```markdown
- Do not silently discard local changes.
```

with:

```markdown
- Do not silently discard local changes.
- Local changes whose paths match `.gitignore` are allowed for `/dp-dev` and `/dp-prod`; report them as ignored-local changes and do not delete, reset, stash, or overwrite them.
- Local changes that do not match `.gitignore` must still block deployment.
```

- [ ] **Step 2: Update global skill safety rules**

Apply the exact same text replacement to `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`.

- [ ] **Step 3: Verify skill text**

Run:

```bash
grep -n "ignored-local\\|gitignore\\|must still block" .trae/skills/project-deploy/SKILL.md
grep -n "ignored-local\\|gitignore\\|must still block" /Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md
```

Expected: both files contain the new safety rules.

- [ ] **Step 4: Commit project skill doc**

Stage only the project skill file:

```bash
git add .trae/skills/project-deploy/SKILL.md
git diff --cached --name-only
git commit -m "docs: update deploy skill ignored changes rule"
```

Expected staged file:

```text
.trae/skills/project-deploy/SKILL.md
```

---

### Task 3: Final Verification

**Files:**
- Verify: `scripts/deploy-dev.sh`
- Verify: `scripts/deploy-prod.sh`
- Verify: `.trae/skills/project-deploy/SKILL.md`

- [ ] **Step 1: Run syntax checks**

Run:

```bash
bash -n scripts/deploy-dev.sh
bash -n scripts/deploy-prod.sh
```

Expected: both exit `0`.

- [ ] **Step 2: Verify current ignored change remains uncommitted**

Run:

```bash
git status --short --branch
git check-ignore --no-index -v frontend/admin/dist/index.html
```

Expected:

```text
 M frontend/admin/dist/index.html
```

Expected: `git check-ignore` reports `.gitignore:8:dist/`.

- [ ] **Step 3: Verify deploy-dev restart behavior**

Run:

```bash
scripts/deploy-dev.sh restart
```

Expected: restart proceeds past worktree validation and prints ignored-local changes.

If it fails later due to infrastructure, ports, Docker, Go, or npm, report that as an environment/runtime failure, not a worktree-check failure.

- [ ] **Step 4: Verify local service availability**

Run:

```bash
scripts/deploy-dev.sh verify
```

Expected output includes:

```text
http://localhost:5173 -> 200
http://localhost:5175 -> 200
http://localhost:8080/api/v1/products -> 200
ws://localhost:8083/ws -> 端口已监听
本地验证通过
```

- [ ] **Step 5: Final Git state**

Run:

```bash
git status --short --branch
git log --oneline -5 --decorate
```

Expected:

- New script and skill commits exist.
- `frontend/admin/dist/index.html` remains uncommitted.
- No accidental staging exists.

---

## Self-Review Checklist

- Spec coverage: both `/dp-dev` and `/dp-prod` are covered.
- Safety coverage: `HEAD == origin/main` remains required; non-ignored changes still block.
- Ignored tracked coverage: `git check-ignore --no-index` handles tracked files such as `frontend/admin/dist/index.html`.
- User-change safety: no step deletes, resets, stashes, or commits `frontend/admin/dist/index.html`.
