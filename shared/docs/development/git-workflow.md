# Git Workflow

This document describes the branching strategy, commit conventions, and pull request process for the Real Assessment Platform.

---

## Branching Strategy

We use a simplified **GitHub Flow** with long-lived `main` and optional release branches.

```
main (protected)
  ├── feat/user-profile
  ├── fix/ai-timeout-error
  ├── chore/update-deps
  └── hotfix/security-patch
```

### Branch Naming

```
<type>/<short-description>
```

| Type | Purpose | Example |
|------|---------|---------|
| `feat` | New feature | `feat/oauth-github` |
| `fix` | Bug fix | `fix/interview-status-bug` |
| `refactor` | Code restructuring (no behavior change) | `refactor/extract-score-service` |
| `docs` | Documentation changes | `docs/api-scoring-endpoints` |
| `test` | Test additions or fixes | `test/interview-e2e` |
| `chore` | Maintenance, dependencies, tooling | `chore/update-go-1.21` |
| `perf` | Performance improvements | `perf/db-query-optimization` |
| `ci` | CI/CD pipeline changes | `ci/add-security-scan` |
| `hotfix` | Urgent production fix | `hotfix/auth-token-leak` |

### Branch Rules

- **`main`** is the single source of truth. Always deployable.
- All work is done on feature branches, merged via Pull Request.
- Branch names use lowercase with hyphens (`-`), no underscores.
- Delete branches after merge.

---

## Commit Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Type

| Type | When to Use |
|------|-------------|
| `feat` | New user-facing functionality |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Code style changes (formatting, no logic change) |
| `refactor` | Code restructuring without behavior change |
| `perf` | Performance improvement |
| `test` | Adding or fixing tests |
| `chore` | Build, deps, tooling, non-user-facing changes |
| `ci` | CI/CD configuration changes |
| `revert` | Reverting a previous commit |

### Scope (optional)

The service or area affected:

```
feat(user): add email verification
fix(ai): handle OpenAI rate limiting
docs(api): update scoring endpoints
chore(deps): bump Go version to 1.21
refactor(resume): extract parser to separate package
test(interview): add completion flow integration tests
```

### Description Rules

- Use imperative mood ("add" not "added" or "adds")
- No period at the end
- First letter lowercase
- Maximum 72 characters

### Breaking Changes

Add `!` after type/scope and include `BREAKING CHANGE:` in the footer:

```
feat(api)!: change authentication token format

BREAKING CHANGE: Access tokens now use RS256 instead of HS256.
Old tokens will be rejected.
```

### Examples

```
feat(user): add OAuth registration
fix(scoring): correct weight calculation for resume scores
refactor(interview): simplify status transition logic
docs: add architecture sequence diagrams
chore(python-common): add async database module
test(ai): mock OpenAI responses in integration tests
perf(api-gateway): add response caching for user profiles
ci: add Trivy image scanning to pipeline
hotfix: patch JWT validation bypass
```

---

## Pull Request Process

### 1. Create the PR

```bash
# Push your branch
git push -u origin feat/my-feature

# Create PR (GitHub CLI)
gh pr create --title "feat(user): add profile page" --body "..."
```

### 2. PR Template

All PRs follow the template at `.github/PULL_REQUEST_TEMPLATE.md`:

- **Description:** What does this change do?
- **Type:** feat, fix, refactor, etc.
- **Testing:** What tests were run?
- **Screenshots:** For UI changes
- **Checklist:** Code style, tests, docs updated

### 3. Review Requirements

- **Minimum 1 approval** from a code owner
- **All CI checks must pass** (lint, test, build)
- **No unresolved conversations**
- **Commits are squashed** or rebased cleanly

### 4. Review Guidelines

**Authors should:**
- Keep PRs small (< 400 lines of code changes ideal)
- Write a clear description of the change
- Self-review before requesting reviewers
- Respond to review comments promptly

**Reviewers should:**
- Review within 24 hours of request
- Focus on correctness, security, and clarity
- Suggest improvements, not just find problems
- Use "Request changes" for blocking issues, "Approve" when satisfied

### 5. Merge Strategy

- **Squash and merge** is the default for feature branches
- **Rebase and merge** for hotfixes (preserves individual commits if meaningful)
- Delete the source branch after merge

---

## Working with the Repository

### Setup

```bash
# Clone
git clone https://github.com/your-org/real-assessment.git
cd real-assessment

# Install pre-commit hooks (optional but recommended)
make hooks
```

### Staying Up to Date

```bash
# Update your branch with latest main
git checkout main
git pull origin main
git checkout feat/my-feature
git rebase main

# Resolve conflicts if any, then continue
git rebase --continue
```

### Undoing Changes

```bash
# Undo last commit (keep changes)
git reset --soft HEAD~1

# Undo last commit (discard changes)
git reset --hard HEAD~1

# Revert a merged PR
git revert <commit-hash>
```

### Working with Submodules (if used)

```bash
# Initialize submodules
git submodule update --init --recursive

# Update submodules
git submodule update --remote
```

---

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`

- **MAJOR:** Breaking changes
- **MINOR:** New features (backward compatible)
- **PATCH:** Bug fixes

### Creating a Release

```bash
# Tag the release
git tag -a v1.2.0 -m "Release v1.2.0: Interview management and AI scoring"

# Push tag
git push origin v1.2.0
```

The CI pipeline automatically:
1. Builds and tests the tagged commit
2. Creates a GitHub Release with changelog
3. Deploys to staging
4. Creates a production deployment PR

---

*See also: [Coding Standards](./coding-standards.md) | [Testing Guide](./testing-guide.md)*
