# ADR 0001: Technology Stack & Architectural Baseline

## Status
Accepted

## Context
Commercial SSH management tools like Termius gate core productivity and collaboration features (End-to-End Encrypted Cloud Sync, Team Vaults, Session Logging, Startup Snippets, Terminal Multiplayer, and Cloud Stack Integrations) behind expensive recurring subscription tiers ($10–$30/seat/month).

We require a 100% free, open-source, self-hostable alternative that provides equivalent or superior security, performance, and cross-platform terminal UX with zero paywalls.

## Decision
We chose the following technology baseline:

1. **Backend: Go 1.24 with Clean Architecture (Hexagonal)**
   - **Rationale:** Go delivers high-concurrency network I/O, low memory footprint, and first-class standard library support for SSH (`golang.org/x/crypto/ssh`) and WebSockets.
   - **Structure:** Clean Architecture (`domain` -> `ports` -> `usecase` -> `adapters`) ensures enterprise business logic is completely isolated from HTTP frameworks, database drivers, and external protocols.

2. **Frontend: Svelte 5 (Runes) + Tailwind CSS + shadcn-svelte**
   - **Rationale:** Svelte 5 with fine-grained reactivity (`$state`, `$derived`, `$props`) provides instantaneous terminal UI responsiveness with zero virtual DOM overhead. Ideal for embedding with xterm.js terminal canvases.
   - **Styling:** Tailwind CSS + shadcn-svelte gives a clean, accessible interface matching modern desktop terminal standards.

3. **Database: PostgreSQL 15+ with versioned SQL migrations**
   - **Rationale:** Robust ACID transactions, native UUID, JSONB capabilities for extensible metadata, and strong concurrency.
   - **Discipline:** Versioned `.sql` migrations in `migrations/`, explicit indexing on all Foreign Keys, and strict parameterized queries using `pgx`.

4. **Cryptography: Client-Side E2EE WebCrypto (AES-256-GCM + Argon2id)**
   - **Rationale:** Zero-Knowledge architecture ensures the self-hosted backend only stores ciphertext; compromised servers never expose server credentials or SSH keys.

## Consequences
- **Positive:**
  - High developer velocity and clean testability.
  - Complete freedom from vendor lock-in and zero software licensing cost.
  - Zero-knowledge data privacy.
- **Negative:**
  - Users who lose their master password cannot recover encrypted vault items unless emergency recovery keys are exported.
