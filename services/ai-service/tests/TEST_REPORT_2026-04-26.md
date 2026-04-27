# AI Service Test Report (2026-04-26)

## Scope
- Service-level tests: `tests/test_services.py`
- API-level tests: `tests/test_api.py`
- Interview-specific API subset (next-question + post-analysis)
- Stability checks via repeated runs
- Coverage snapshot for current suite

## Environment
- OS: macOS
- Python: 3.12.13 (venv)
- Pytest: 8.3.4
- Plugins: pytest-cov 6.0.0, pytest-asyncio 0.25.2, anyio 4.13.0

## Test Matrix And Results

### 1) Full regression (services + api)
Command:
`python -m pytest tests/test_services.py tests/test_api.py -q`

Result:
- PASS
- 49 passed
- 1 warning
- Runtime: ~0.87s to ~0.92s depending on run

### 2) Interviewer API subset
Command:
`python -m pytest tests/test_api.py -k interviewer -q`

Result:
- PASS
- 4 passed, 10 deselected
- Runtime: ~0.57s

### 3) Interview communication and post-analysis helpers subset
Command:
`python -m pytest tests/test_services.py -k "InterviewCommunicationHelpers or PostAnalysisCommunicationHelpers" -q`

Result:
- PASS
- 5 passed, 30 deselected
- Runtime: ~0.43s

### 4) Stability checks (repeat interviewer subset 3x)
Command pattern:
`for i in 1 2 3; do python -m pytest tests/test_api.py -k interviewer -q; done`

Result:
- PASS on all 3 runs
- Run 1: 4 passed
- Run 2: 4 passed
- Run 3: 4 passed
- No flaky failure observed

### 5) Coverage-enabled regression snapshot
Command:
`python -m pytest tests/test_services.py tests/test_api.py --cov=src --cov-report=term-missing -q`

Result:
- PASS
- 49 passed, 1 warning
- Total coverage: 57%
- High coverage examples:
  - src/core/prompt_templates.py: 100%
  - src/models/responses.py: 100%
  - src/main.py: 100%
- Lower coverage area:
  - src/api/routes.py: 44%
  - src/core/embeddings.py: 19%

## Conditions Verified
- Normal path: full test suite pass
- Weak interview answer path: covered in API and helper tests
- Strong interview answer path: covered via deterministic API test
- Empty conversation path: post-analysis returns zeroed scores and explanatory message
- LLM failure path: next-question fallback behavior verified
- Repeatability: interviewer subset stable across 3 consecutive runs

## Warnings And Non-Blocking Findings
- Repeated warning from pytest-asyncio:
  - `asyncio_default_fixture_loop_scope` is unset
- Additional warning from a legacy test style:
  - `DeprecationWarning: There is no current event loop` in `tests/test_services.py`

## Risk Assessment
- Functional risk (tested scope): LOW
- Flakiness risk for interviewer routes: LOW (no flakiness observed in repeated runs)
- Coverage risk: MEDIUM (total 57%, `src/api/routes.py` still has significant untested branches)

## Recommended Next Actions
1. Set `asyncio_default_fixture_loop_scope` in `pyproject.toml` to remove pytest-asyncio deprecation warning.
2. Migrate legacy event-loop usage (`asyncio.get_event_loop().run_until_complete`) to modern async test patterns.
3. Expand tests for untested `src/api/routes.py` branches (especially long fallback/guardrail branches).
4. Add targeted tests for `src/core/embeddings.py` to raise confidence in semantic-duplicate and similarity behavior.
