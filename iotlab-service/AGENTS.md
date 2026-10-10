# AGENTS.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 0. Project Guideline

**Read `docs/GUIDELINE.md` before implementing any feature.**

When coding a Go feature in this repository, also read:

- `.agents/skills/go-style-core/SKILL.md`
- `.agents/skills/go-clean-cqrs-architecture/SKILL.md`

`docs/GUIDELINE.md` is the authoritative reference for this template. It covers:

- Full directory structure and layer contracts
- Exact patterns for models, DTOs, commands, queries, handlers, workers, repositories
- Step-by-step checklist to implement a new domain end-to-end
- Which files to delete and which to keep when replacing the sample `user` domain
- Config, error handling, and response conventions

Do not implement a feature, add a layer, or create a new domain before reading the relevant section.

---

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Validate Once at the Right Layer

**Validate at the entry boundary. Trust layers below don't need to repeat it.**

- The layer that first receives external input owns validation. For HTTP handlers that means the DTO + Echo validator.
  For consumers it means the message deserialization step.
- **Application/service layers must NOT re-validate** what the caller already validated (nil checks, zero-value guards,
  required-field assertions). These checks add noise and imply distrust of the contract.
- Only add validation in a deeper layer when there is **genuine business logic** that layer uniquely owns — e.g. a
  cross-field invariant, an external-system constraint, or a rule that no caller can pre-check.
- Before adding any guard/check, ask: "has the caller already ensured this?" If yes, delete the check.

Rule of thumb per layer in this project:

| Layer | Validates |
| --- | --- |
| Handler | DTO binding + struct tags (required, min, format…) |
| App | Business invariants only — never nil/zero-value re-checks |
| Repository | Nothing — trusts the app layer |

## 5. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and
clarifying questions come before implementation rather than after mistakes.
