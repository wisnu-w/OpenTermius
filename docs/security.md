# OpenTermius Security & Trust Model

This document outlines the threat model, cryptography standards, trust boundaries, and authorization policies of OpenTermius.

## 1. Concrete Trust Boundaries

```
[ Untrusted Client Device ]
           │
           │ TLS 1.3 + JWT Auth
           ▼
┌────────────────────────────────────────────────────────┐
│ Boundary 1: Perimeter API Gateway / Reverse Proxy     │
│ - Strict TLS enforcement                               │
│ - Rate limiting (10 req/s login, 100 req/s API)        │
│ - Input validation & request size limits (max 5 MB)    │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│ Boundary 2: Application Layer (Go Backend)             │
│ - JWT validation & Claims parsing                      │
│ - Role-Based Access Control (RBAC: Owner/Admin/Member) │
│ - ZERO-KNOWLEDGE: Stores ciphertext blobs only         │
│ - Parameterized SQL (pgx) with zero raw string queries │
└──────────────────────────┬─────────────────────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│ Boundary 3: Database Layer (PostgreSQL)                │
│ - Dedicated non-superuser DB user                      │
│ - Foreign Key integrity + cascade deletion rules       │
│ - Encrypted-at-rest volumes                            │
└────────────────────────────────────────────────────────┘
```

## 2. Zero-Knowledge End-to-End Encryption (E2EE)

### Cryptographic Primitive Selection
- **Symmetric Encryption:** AES-256-GCM (Galois/Counter Mode) with a unique 96-bit (12-byte) initialization vector (IV / nonce) per item.
- **Key Derivation (KDF):** Argon2id (`time=3`, `memory=64MB`, `parallelism=4`) derived from user master password with unique user salt.
- **Key Wrapping:** Team Vault Master Keys are encrypted using AES-KeyWrap (or RSA-OAEP / ECDH P-256) using each authorized member's public key.

### Zero-Knowledge Guarantee
The server never receives:
1. The plaintext master password.
2. The decrypted vault keys.
3. The plaintext hostnames, IP addresses, credentials, private SSH keys, or passwords.
All encryption and decryption occurs in client memory using WebCrypto APIs (`window.crypto.subtle`) or desktop native keyrings (`keytar`).

## 3. Terminal & SSH Security

- **Direct vs Relayed Connection:** By default, OpenTermius client connects directly to target SSH servers whenever the target is network-routable.
- **WebSocket Gateway:** When proxying via the self-hosted backend, the connection utilizes Go's `golang.org/x/crypto/ssh` implementation with strict host key verification.
- **No Command Injection:** SSH command invocation and shell allocation strictly use pseudo-terminal handles (`node-pty` / Go PTY) without string concatenation through `/bin/sh -c`.
- **Session Multiplayer Authorization:** Only authorized vault members can join shared live terminal sessions (`/ws/sessions/:id/join`). The host can terminate live multiplayer sharing at any time.

## 4. Anti-Vulnerability Verification Checklist

- [x] **No SQL Injection:** All queries use `$1, $2` parameters via `pgx`.
- [x] **No IDOR (Insecure Direct Object Reference):** Every vault item query explicitly joins or filters against `vault_id` and user membership (`vault_members`).
- [x] **No Hardcoded Secrets:** Zero hardcoded API keys, secrets, or default passwords.
- [x] **CSRF / CORS Protection:** Explicit origin validation for WebSockets and API endpoints.
- [x] **Rate Limiting:** Authentication endpoints are rate-limited to prevent brute-force attacks.
