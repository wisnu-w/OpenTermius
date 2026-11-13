# OpenTermius Architecture Guide

This document defines the architectural layers, system topology, component interactions, and data flow of OpenTermius.

## 1. Concentric Layer Structure (Hexagonal / Clean Architecture)

Dependencies point strictly inward:

```
┌─────────────────────────────────────────────────────────────┐
│ Adapters Layer (External)                                   │
│  - HTTP Handlers (Gin/Chi)      - Postgres Repositories     │
│  - WebSocket Terminal Bridges   - OS Native / Cloud APIs    │
├─────────────────────────────────────────────────────────────┤
│ Ports Layer (Contracts)                                     │
│  - Inbound: AuthUseCase, VaultUseCase, TerminalUseCase      │
│  - Outbound: UserRepository, VaultRepository, SessionStore  │
├─────────────────────────────────────────────────────────────┤
│ UseCase Layer (Application Business Rules)                  │
│  - User registration & team management                      │
│  - E2EE vault synchronization and conflict resolution       │
│  - SSH connection proxying & multiplayer session broadcast  │
├─────────────────────────────────────────────────────────────┤
│ Domain Entities (Core Enterprise Logic)                     │
│  - User, Vault, VaultItem (Host, Key, Snippet, Tunnel)      │
│  - Zero external framework dependencies                     │
└─────────────────────────────────────────────────────────────┘
```

## 2. Directory Structure & Layer Responsibilities

```
backend/
├── cmd/
│   └── api/
│       └── main.go                 # Composition Root: Dependency Injection & bootstrap
└── internal/
    ├── domain/                     # Pure Enterprise Entities
    │   ├── entity.go               # Struct definitions (Host, Vault, Key, Snippet)
    │   └── errors.go               # Standard Domain error definitions
    ├── ports/                      # Layer Boundaries & Contracts
    │   ├── services.go             # Inbound interfaces (called by handlers)
    │   ├── repositories.go         # Outbound interfaces (called by use cases)
    │   └── ssh.go                  # SSH service and session contracts
    ├── usecase/                    # Application Business Rules
    │   ├── auth_usecase.go         # Login, register, token issuance
    │   ├── vault_usecase.go        # Vault CRUD, sync, versioning
    │   └── terminal_usecase.go     # SSH connection session orchestration
    └── adapters/                   # Infrastructure Implementations
        ├── handler/
        │   └── http/               # Transport: HTTP & WebSocket handlers
        │       ├── routes.go       # Route definitions
        │       ├── auth_handler.go # REST handlers for auth
        │       ├── vault_handler.go# REST handlers for vaults & items
        │       └── terminal_ws.go  # WebSocket duplex SSH stream
        ├── repository/
        │   └── postgres/           # Persistence: Database access via pgx
        │       ├── user_repository.go
        │       └── vault_repository.go
        └── ssh/                    # SSH PTY Engine (golang.org/x/crypto/ssh)
            └── client.go           # Real SSH client, PTY, password & key auth
```

## 3. Communication Flows

### A. Vault Synchronization Flow
1. Client encrypts item payload with AES-GCM-256 using the local Vault Key.
2. Client sends encrypted ciphertext and item metadata to `POST /api/v1/vaults/:id/items`.
3. `VaultHandler` parses DTO -> calls `VaultService.CreateItem(ctx, item)`.
4. `VaultUseCase` validates membership and team permissions -> calls `VaultRepository.SaveItem(ctx, item)`.
5. `VaultRepository` writes to PostgreSQL `vault_items` table with parameterized SQL.
6. Server broadcasts a sync notification event to other active clients of that vault.

### B. Interactive SSH Terminal Flow
1. Client establishes WebSocket connection to `/ws/terminal/:host_id`.
2. Server validates user JWT and decrypts client SSH parameters if provided or requests credentials.
3. Server SSH bridge establishes connection to target SSH server.
4. Input and output buffers are streamed full-duplex through the WebSocket to xterm.js in the browser/desktop.
5. If multiplayer sharing is enabled, additional clients joining `/ws/sessions/:id/join` receive replicated terminal stdout.

## 4. Error Handling Strategy

Standard domain errors map deterministically to HTTP response codes:
- `domain.ErrNotFound` -> `404 Not Found`
- `domain.ErrUnauthorized` -> `401 Unauthorized`
- `domain.ErrForbidden` -> `403 Forbidden`
- `domain.ErrConflict` -> `409 Conflict`
- `domain.ErrInvalidInput` -> `400 Bad Request`
- Unhandled/Internal -> `500 Internal Server Error` (with sanitized public message)

## 5. Multiplatform Client Topology (Tauri v2 + Svelte 5)

OpenTermius uses a unified multiplatform architecture targeting **macOS, Windows, Linux, and Android**:

```
opentermius/
├── backend/                       # Go 1.24 Clean Architecture Sync & SSH Streaming Gateway
│   ├── internal/domain/           # Entities (User, Vault, VaultItem)
│   ├── internal/ports/            # Contracts (SSH, Vault, Auth, Logs)
│   └── internal/adapters/         # HTTP/WS handlers, Postgres repository
├── client-app/                    # Multiplatform Client App (Desktop macOS/Win/Linux, Mobile Android)
│   ├── src/                       # Svelte 5 (Runes), xterm.js terminal, E2EE WebCrypto
│   └── src-tauri/                 # Tauri v2 Multiplatform Core (Rust In-Memory PTY, OS Keychain)
│       ├── tauri.conf.json        # Bundle identifiers, window specs, permissions
│       └── src/lib.rs             # Native PTY session manager (zero listening ports)
└── gateway-web/                   # Gateway Admin & Team Management Portal (Web)
    └── src/                       # Svelte 5 + Tailwind CSS cluster overview, audit logs, team invites
```

### Operational Modes:
- **Connect to Gateways (Cloud / Team Sync)**: Client connects to self-hosted Go backend via custom REST API URL (`http://<ip>:8080/api/v1`) and WebSocket bridge (`ws://<ip>:8080/ws`). Synchronizes vaults, items, and audit logs with remote PostgreSQL.
- **Standalone Mode (Local-First Offline - Zero Open Ports)**: Client runs 100% locally with zero network requirement. All vault items, keys, and configurations are encrypted locally using WebCrypto (AES-256-GCM) in secure browser/desktop storage. In desktop apps, SSH sessions run via Native In-Memory PTY IPC (`portable-pty`) directly to the target SSH server without opening any local listening ports on the machine.

### Targets & Packaging:
- **macOS Desktop**: Bundled via Tauri into native `.app` and `.dmg` (`aarch64` Apple Silicon & `x86_64` Intel).
- **Android Mobile**: Ready for `npx tauri android init` and APK bundling once Android SDK/NDK is configured.
- **Port Forwarding & Proxies**:
  - **Jump Host (Bastion)**: ProxyJump SSH chaining between hosts.
  - **Dynamic SOCKS5**: Local SOCKS5 proxy server routed through SSH (`ssh -D`).
  - **Local Port Forwarding**: Direct TCP mapping (`127.0.0.1:localPort -> remoteHost:remotePort`).
  - Stored in Supabase PostgreSQL under `vault_items` table with `kind: 'tunnel'`.

### 5. SFTP File Manager Subsystem
- **Backend Architecture**:
  - Implements `ports.SFTPService` via `github.com/pkg/sftp` on top of standard Go `crypto/ssh`.
  - Supports directory listing with metadata (size, modTime, POSIX permissions), file streaming downloads, multipart uploads, folder creation, and recursive/single item deletion.
  - Endpoints protected under `/api/v1/sftp/*` with JWT and standalone token compatibility.
- **Frontend File Explorer (`SftpView.svelte`)**:
  - Simple Termius-style interface: host picker, breadcrumb navigation, parent folder jump (`..`), search filter.
  - File table with icons (archive, code, folder, document), formatted byte sizes, download/delete actions, upload picker, and folder creation modal.
  - One-click access from any Host Card (`SFTP` button) or the Sidebar/Bottom Nav.

### 6. Multiplatform Responsive Design (Phone, Tablet, Laptop)
- **Mobile Handheld (<768px)**:
  - **Slide-over Drawer**: Sidebar converts into off-canvas drawer with backdrop overlay.
  - **Bottom Navigation Bar**: Fixed thumb-friendly bottom nav (`Hosts`, `Terminal` with active tab badge, `Tunnels`, `Vaults`, `Menu`).
  - **Virtual Touch Accessory Bar**: Terminal on-screen keys (`ESC`, `TAB`, `^C`, `^D`, `^Z`, arrows `↑ ↓ ← →`, `/`, `|`, `~`, `-`, `:`) with toggle button.
  - **Auto-Stack Split Terminal**: Side-by-side terminal splits gracefully fall back to stacked vertical panes (`flex-col md:flex-row`) to prevent unreadable narrow columns.
  - **Touch-Friendly Modals**: All dialogs use `max-h-[90vh] overflow-y-auto` with touch-sized tap targets.
- **Tablet (768px - 1024px)**:
  - Collapsible persistent sidebar with keyboard shortcut (`⌘B`).
  - 2-column responsive host grid (`sm:grid-cols-2`).
- **Laptop & Desktop (>1024px)**:
  - Full persistent multi-pane terminal split (Horizontal / Vertical).
  - 3-4 column host cards grid (`lg:grid-cols-3 xl:grid-cols-4`).
