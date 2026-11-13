# OpenTermius

OpenTermius is a 100% free, self-hostable, open-source alternative to Termius. It provides modern SSH terminal management, end-to-end encrypted personal and team vaults, real-time terminal multiplayer, startup snippet automation, and session logging without any commercial paywalls or subscriptions.

## Features
- **Zero Paywalls:** All premium features (Cloud sync, Team Vaults, Session Recording, Snippets, Cloud VM discovery) free and built into core.
- **End-to-End Encryption (E2EE):** Client-side zero-knowledge encryption with AES-GCM-256. The server never sees raw private keys or credentials.
- **Real-time Terminal Multiplayer:** Share interactive live terminal sessions with teammates via WebSockets.
- **Port Forwarding & Proxies:** Local Port Forwarding, Remote Tunnels, Dynamic SOCKS5 Proxy, and Jump Host (Bastion) chaining.
- **Keychain & Identities:** Reusable SSH keys (ED25519/RSA-4096 generator + PEM import) and credential profiles.
- **Cross-Platform Sync:** Self-hosted sync server with PostgreSQL backend.
- **Native Multiplatform:** Lightweight (<10MB) desktop binaries for macOS, Windows, Linux, and Android via Tauri v2.
- **Hardware Keys:** FIDO2 / WebAuthn & biometric key generation.
- **Session Logs & Audit Trails:** Searchable terminal logs and session playback.

## Architecture & Workspace Structure
- **`backend/`**: Go 1.24 Clean Architecture (Hexagonal: `domain` -> `ports` -> `usecase` -> `adapters`). REST API & WebSocket streaming server.
- **`client-app/`**: Multiplatform client (macOS, Windows, Linux, Android) via Tauri v2 + Svelte 5 (Runes). Supports **Connect to Gateways** (multi-device sync) and **Standalone Mode** (local-first, zero-port native in-memory PTY SSH).
- **`gateway-web/`**: Web-based Gateway Admin & Team Management Portal (Svelte 5 + Tailwind CSS) for cluster health, team memberships, audit logs, and gateway endpoint control.
- **`migrations/`**: Versioned PostgreSQL DDL schema & indexes.

## Quickstart

### Prerequisites
- Go 1.24+
- Node.js 22+ & npm
- PostgreSQL 15+ (or Supabase connection via `.env`)

### 1. Configure Environment
```bash
cp .env.example .env
```

### 2. Run Database Migrations
```bash
# Using migrate CLI or psql
psql -h localhost -U postgres -d opentermius -f migrations/000001_init.up.sql
```

### 3. Run Backend API & WebSocket Server
```bash
cd backend
go run cmd/api/main.go
# Server listens on http://localhost:8080
```

### 4. Run Client Application (Desktop / Web)
```bash
cd client-app
npm install
npm run dev
# Running on http://127.0.0.1:5173
```

### 5. Run Gateway Web Admin Portal
```bash
cd gateway-web
npm install
npm run dev
# Running on http://127.0.0.1:5174
```

---

## Multiplatform Desktop & Mobile Client (`client-app/`)

OpenTermius compiles into lightweight native binaries (<10MB installer) with native in-memory PTY (zero listening ports) and OS Keychain integration.

### Modes of Operation
1. **Connect to Gateways:** Connects to self-hosted OpenTermius backend cluster for real-time E2EE vault sync, team sharing, and session logs.
2. **Standalone Mode:** Local-first offline mode. Private keys and configurations are encrypted client-side using WebCrypto AES-256-GCM in OS local sandbox. Direct outbound SSH connection via native in-memory PTY without opening any local TCP listening ports.

### Prerequisites for Native Builds
- **Rust Stable:** `rustup default stable`
- **Node.js 22+**
- *(Optional for Android)*: Android Studio, Android SDK (`ANDROID_HOME`), and NDK (`NDK_HOME`)

### 1. Run Desktop in Development Mode (macOS / Windows / Linux)
```bash
cd client-app
npm run tauri:dev
```

### 2. Build macOS Desktop App (`.dmg` & `.app`)
```bash
cd client-app
npm run tauri:build
```
Output bundles:
- **DMG Installer:** `client-app/src-tauri/target/release/bundle/dmg/OpenTermius_0.1.0_aarch64.dmg` (~3.0 MB)
- **App Binary:** `client-app/src-tauri/target/release/bundle/macos/opentermius.app`

Untuk membuka installer DMG:
```bash
open client-app/src-tauri/target/release/bundle/dmg/OpenTermius_0.1.0_aarch64.dmg
```

### 3. Build Android Mobile App (`.apk`)
```bash
cd client-app
# Initialize Android project structure (first time only)
npx tauri android init

# Build debug APK
npx tauri android build --debug

# Or build release APK / AAB
npx tauri android build
```
Output APK:
- `client-app/src-tauri/gen/android/app/build/outputs/apk/debug/app-debug.apk`

---

## Documentation
- [System Architecture](docs/architecture.md)
- [Database Schema & Indexes](docs/database.md)
- [API & WebSocket Specifications](docs/api.md)
- [Security & Cryptography Model](docs/security.md)
- [Architecture Decision Records](docs/decisions/)
