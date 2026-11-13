# Project Context & AI Rules: OpenTermius

Open-source, self-hostable, zero-paywall alternative to Termius for desktop and web.
Includes E2EE Personal & Team Vaults, multi-device sync, session logs, startup snippets automation, and terminal multiplayer.

## 1. Tech Stack & Architecture
- **Backend:** Go 1.24 Clean Architecture (Hexagonal: `domain` -> `ports` -> `usecase` -> `adapters`).
- **Frontend:** Svelte 5 (Runes: `$state`, `$derived`, `$props`, `$effect`) + Tailwind CSS + shadcn-svelte.
- **Database:** PostgreSQL with versioned `.sql` migrations in `migrations/`. Client-side E2EE using WebCrypto (AES-256-GCM).
- **Communication:** REST API for resource management, WebSockets for interactive SSH PTY streaming & multiplayer terminal sharing.

## 2. Anti-Duplicate & Pre-Flight Check (STRICT)
Before adding ANY new function, endpoint, route, or UI component:
1. **Read `docs/api.md` first:** Check if the route or similar endpoint already exists.
2. **Search existing code:** Check existing symbols and handlers before writing new ones.
3. **Reuse or Extend:** If an endpoint/function exists that covers >70% of the need, refactor or extend it. NEVER create parallel/duplicate routes (e.g. creating `/api/v1/vaults/list` when `/api/v1/vaults` exists).

## 3. Documentation Maintenance (Definition of Done)
`docs/` is the living single source of truth. You are FORBIDDEN from finishing a task that changes code without updating documentation:
- **API Change:** When adding/modifying endpoints, update `docs/api.md`.
- **Database Change:** When adding/altering tables or indexes, update `docs/database.md`.
- **Architecture Change:** When introducing new layers or packages, update `docs/architecture.md`.
- **Architectural Decision:** When choosing a new library or structural pattern, add an ADR in `docs/decisions/`.

## 4. Production Discipline & Zero Dummy Data
- **Zero Dummy Data:** Never invent `const mockData = [...]` or return hardcoded mock responses in handlers (`gin.H{"dummy": ...}`) or in-memory slice DBs. Handlers call Services; Services call Repositories; Repositories query PostgreSQL.
- **All 4 UI States:** Frontend components must explicitly cover Loading, Error, Empty, and Success states.
- **Explicit SQL:** No `SELECT *` in backend queries; enumerate columns explicitly. Every Foreign Key column must have an explicit `CREATE INDEX idx_...`.
- **Svelte 5 Only:** Default to Svelte 5 Runes (`$state`, `$derived`, `$props`, `$effect`, snippets). Never use deprecated Svelte 4 syntax (`export let`, `$:`) unless explicitly asked.

## 5. Security-First Baseline (security-audit)
- **Concrete Trust Boundaries:** Public API vs Authenticated User vs Team Admin vs System. Every endpoint validates input strictly.
- **Zero Secrets:** Zero hardcoded secrets, tokens, or credentials in codebase or git. All secrets managed via `.env`.
- **Zero-Knowledge Encryption:** Vault secrets (private keys, passwords, connection configs) are encrypted on client-side before transmission. The server never holds vault master keys.
- **Defensive Safeguards:** Strict input validation, parameterized SQL queries (no string concatenation), rate limiting on authentication and terminal connection initiation.
- **Keep `docs/security.md` updated:** Maintain documented threat models, trust boundaries, and cipher configurations.
