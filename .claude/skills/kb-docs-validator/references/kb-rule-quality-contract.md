# KB rule.md Quality Contract

This contract is the shared source of truth for `kb-init-docs`, `kb-update-docs`, and `kb-docs-validator`.

## Document Ownership

- `CLAUDE.md` owns the compact module entry summary: purpose, responsibilities, key classes, references, and short module structure summaries.
- `docs/rule.md` owns only enforceable future-change constraints for the current doc root.
- Module structure may be mentioned in `docs/rule.md` only when it becomes a constraint, not as a directory inventory.

## Keep-Worthy Rule Shape

Each keep-worthy rule should be a local scenario block:

```md
## <Scenario Name>
WHEN <specific local scenario>:
- MUST <required behavior with local boundary or exception>.
- MUST NOT <forbidden behavior with local boundary or exception>.
```

Rules must be backed by local evidence from the doc root, such as code patterns, public APIs, ownership boundaries, lifecycle/state-machine constraints, dependency constraints, or repeated local anti-patterns.

## Parent Rule Deduplication

Before creating or updating `docs/rule.md`, inspect ancestor rule docs such as `../docs/rule.md`, higher `docs/rule.md` files, or legacy ancestor `rule.md` files.

Do not duplicate parent/global rules in a child doc root. A child `docs/rule.md` should contain only:

- More specific local constraints.
- Local exceptions to a parent rule.
- Local examples that clarify how a parent rule applies in this doc root.

If the parent rule already covers the behavior, reference it from `CLAUDE.md` or keep the child rule omitted. Do not restate the same scenario heading, `WHEN` trigger, or generic `MUST` actions.

## Code Examples

Code examples are allowed when they are compact positive/negative examples attached to a nearby local rule.

Prefer keeping 1-3 high-value examples in a rule doc when the implementation pattern is non-obvious and the example materially improves future coding accuracy, such as:

- Fallback resolution order.
- DI/context propagation.
- Lifecycle/unmount ordering.
- Registration/config mapping.
- State-machine guards.

Keep examples short: usually 5-12 lines each, using real code from the repository. Introduce examples with `Example from <repo-relative-or-doc-root-relative-path>:` or `Bad example from <path>:` under the relevant `WHEN` block. Do not invent APIs, classes, or code snippets that do not exist in the repo.

Do not use `docs/rule.md` for standalone tutorials, long templates, generic style guides, or full implementation walkthroughs.

## Content That Does Not Belong In rule.md

Do not put these in `docs/rule.md` unless they are narrowed into an enforceable local constraint:

- Generic coding advice or platform-common knowledge.
- Pure architecture descriptions.
- Directory/module structure inventories.
- Naming inventories.
- Test matrices.
- Dependency inventories.
- One-off product notes.
- Speculative guidance.
- Rules already covered by global rules.
- Vague advice such as "write clean code", "handle errors", or "pay attention to performance".
- Generic performance, stability, robustness, or reliability advice that is not narrowed into a local constraint.

## Scoped Length Budgets

Classify doc roots deterministically and explainably:

- Small/leaf doc root: no child doc roots, usually a single feature, component, service, or similarly narrow local unit.
- Medium/module doc root: local layered layout, one child doc root, many code files, or many direct subdirectories.
- Large/aggregate doc root: multiple child doc roots or aggregate path names such as `Component`, `Foundation`, `LiveRoom`, `Service`.

Length budgets:

| Scope | Target | Warn | Strong / Error |
| --- | ---: | ---: | ---: |
| Small/leaf | <= 80 lines | > 120 lines | > 200 lines strong warning |
| Medium/module | <= 120 lines | > 180 lines | > 300 lines strong warning |
| Large/aggregate | <= 180 lines | > 250 lines | > 500 lines hard error |

`docs/rule.md` over 500 lines is always a hard quality error.
