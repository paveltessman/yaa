---
name: implement
description: Build a feature in reviewed steps. Plan first, split the work by commits, then write one commit at a time and stop for review. Use when the user asks to implement a feature, a PR of a plan document, or a change described in text, and wants to review each step before it lands.
---

# implement

Three phases. The user reviews between every one of them. You never commit.

```
1. Plan      you read, you ask, you propose commits   -> user approves
2. Build     you write one commit, you verify, stop   -> user reviews and commits
3. Resume    you read HEAD, you build the next one    -> repeat until done
```

## Phase 1: plan

The user names the work as text, or points at a document such as an issue or a plan file.

Read before you plan:

- The document the user named, and the design document above it.
- The code the change touches. Read whole files, not excerpts.
- The nearest thing that already works.
- The tests of that nearest thing. They state the conventions.

Then write the plan in the chat. It holds:

1. **What the change adds**, in four or five lines. Name the layers it touches.
2. **The decisions you made**, with the reason. A design document leaves gaps. Fill them, state that you filled them, and let the user correct you before any code exists.
3. **The commits.** One heading each, with the files and the tests. Say what makes each one green on its own.
4. **What stays out**, and why.

Split the commits so that each one builds and passes on its own.

Ask a question only where two readings lead to different work. Anything else is a decision you state in the plan.

**Stop. Wait for the user to approve the plan.**

## Phase 2: build one commit

Work the current commit and nothing else. Do not start the next one.

Before you stop, every one of these passes:

- `make check` runs tidy, vet, tests and the linter. Run it, or run its parts.
- Export `TEST_DATABASE_URL` before the tests. The database tests fail without it. Run `docker compose up db` first.
- `gofmt -l` on every directory you touched. The linter also runs goimports.

Then report in the chat:

- **What to look at**: a decision worth a second opinion, a trade-off, an assumption.
- **What you had to change outside the commit.** A later commit that reworks a handler breaks the tests of an earlier one. Fix them and say so.

**Stop. Do not run `git commit`. The user commits.**

## Phase 3: resume

The user commits by hand, and may introduce changes while doing it. Before the next commit:

- Read `git log --oneline -3` and `git show --stat HEAD`.
- Take this as the current state. Never silently revert it. If the change is bad, stop and tell the user about it.

## Rules

- **Never commit, never push.** The user does both.
- **One commit per turn.** Green at the end of each.
- **Match the code around you.** Comment density, naming, error wording, test style. The nearest working file is the specification.
- **Write the comments and the chat in ASD-STE100.** The user rule set applies. Code and identifiers do not.
- **Say when a test is weak.** A test that passes because a fake returns the zero value proves nothing. Scope it to the part under test.

## Notes

- Migrations are goose SQL files under `migrations/sql`, embedded by `migrations/migrations.go`. Scaffold with `make migration name=add_reminders`. squawk lints them in pre-commit.
- Database tests read `TEST_DATABASE_URL` and fail without it. They must roll back what they apply.
- Settings come from the environment through `platform/settings`, with a default for everything but `TG_TOKEN`. Add a new key to `.env.example` too.
