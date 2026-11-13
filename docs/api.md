# OpenTermius API Specification

This document serves as the single source of truth for all OpenTermius REST API endpoints and WebSocket channels.

## Base URL & Authentication
- Base URL: `http://localhost:8080/api/v1`
- WebSocket Base URL: `ws://localhost:8080/ws`
- Auth Header: `Authorization: Bearer <jwt_token>` (for REST) or query param `?token=<jwt_token>` (for WebSockets).

---

## Endpoint Catalog

| Method | Path | Summary | Auth Required |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | Register new user account | No |
| `POST` | `/api/v1/auth/login` | Login and receive JWT access token | No |
| `GET` | `/api/v1/auth/me` | Fetch authenticated user profile | Yes |
| `GET` | `/api/v1/vaults` | List accessible personal & team vaults | Yes |
| `POST` | `/api/v1/vaults` | Create a new personal or team vault | Yes |
| `GET` | `/api/v1/vaults/:id` | Get vault details & wrapped keys | Yes |
| `GET` | `/api/v1/vaults/:id/items` | List encrypted items in vault | Yes |
| `POST` | `/api/v1/vaults/:id/items` | Add encrypted item to vault | Yes |
| `PUT` | `/api/v1/vaults/:id/items/:itemId` | Update encrypted item | Yes |
| `PUT` | `/api/v1/vaults/:id/items/:itemId/move` | Move item to another vault | Yes |
| `DELETE`| `/api/v1/vaults/:id/items/:itemId` | Delete item from vault | Yes |
| `GET` | `/api/v1/vaults/:id/members` | List members of a team vault | Yes |
| `POST` | `/api/v1/vaults/:id/members` | Invite a teammate to vault by email | Yes |
| `GET` | `/api/v1/session-logs` | List recorded terminal session logs | Yes |
| `POST` | `/api/v1/session-logs` | Save completed terminal session log | Yes |
| `WS` | `/ws/terminal/:hostId` | Interactive SSH terminal session stream | Yes |
| `WS` | `/ws/sessions/:sessionId/join` | Join terminal multiplayer live session | Yes |

---

## Detailed Request & Response Schemas

### 1. `POST /api/v1/auth/register`
**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "name": "Jane Doe"
}
```
**Response (201 Created)**:
```json
{
  "user": {
    "id": "8a004dbc-b8c1-45c8-9cc6-32189cd9d327",
    "email": "user@example.com",
    "name": "Jane Doe",
    "created_at": "2026-11-12T11:00:00Z"
  },
  "token": "eyJhbGciOi..."
}
```

### 2. `POST /api/v1/auth/login`
**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```
**Response (200 OK)**:
```json
{
  "user": {
    "id": "8a004dbc-b8c1-45c8-9cc6-32189cd9d327",
    "email": "user@example.com",
    "name": "Jane Doe"
  },
  "token": "eyJhbGciOi..."
}
```

### 3. `POST /api/v1/vaults/:id/items`
**Request Body**:
```json
{
  "kind": "host",
  "label": "Production Web Server",
  "ciphertext": "eyJpdiI6IlpB...",
  "nonce": "aW5pdGlhbGl6YXRpb252ZWN0b3I="
}
```
**Response (201 Created)**:
```json
{
  "id": "b3e04746-fc93-4a0d-b4cb-7dca551e7f60",
  "vault_id": "8a004dbc-b8c1-45c8-9cc6-32189cd9d327",
  "kind": "host",
  "label": "Production Web Server",
  "ciphertext": "eyJpdiI6IlpB...",
  "nonce": "aW5pdGlhbGl6YXRpb252ZWN0b3I=",
  "version": 1,
  "created_at": "2026-11-12T11:05:00Z",
  "updated_at": "2026-11-12T11:05:00Z"
}
```

### 4. `WS /ws/terminal/:hostId?token=<jwt_token>`
**WebSocket Protocol Messages**:
- **Client -> Server Handshake (First message on connect)**:
  ```json
  {
    "type": "init",
    "host": "54.210.88.14",
    "port": 22,
    "username": "ubuntu",
    "authType": "password",
    "password": "SecretPassword123!",
    "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
    "passphrase": "",
    "cols": 80,
    "rows": 24
  }
  ```
- **Client -> Server (Keystrokes & Control)**:
  - Raw interactive keystrokes: plain text / binary bytes forwarded directly to SSH PTY stdin.
  - Window resize event:
    ```json
    {"type": "resize", "cols": 120, "rows": 36}
    ```
- **Server -> Client (PTY Output Stream)**:
  - Real-time ANSI colored binary / text stream from remote SSH PTY stdout/stderr, written directly to xterm.js.

### 5. `POST /api/v1/sftp/list`
List files and directories on remote host via SFTP.
**Request**:
```json
{
  "host": "192.168.1.10",
  "port": 22,
  "username": "root",
  "authType": "password",
  "password": "secret",
  "path": "/var/www"
}
```
**Response (200 OK)**:
```json
{
  "path": "/var/www",
  "items": [
    {
      "name": "html",
      "path": "/var/www/html",
      "isDir": true,
      "size": 4096,
      "modTime": "2026-11-12 10:00:00",
      "permissions": "drwxr-xr-x"
    }
  ]
}
```

### 6. `POST /api/v1/sftp/download`
Stream remote file download.
**Request**:
```json
{
  "host": "192.168.1.10",
  "port": 22,
  "username": "root",
  "authType": "password",
  "password": "secret",
  "path": "/var/log/nginx/access.log"
}
```
**Response (200 OK)**:
- Binary octet-stream with `Content-Disposition: attachment; filename="access.log"`.

### 7. `POST /api/v1/sftp/upload`
Upload file via multipart/form-data.
**Fields**: `host`, `port`, `username`, `authType`, `password`, `privateKey`, `passphrase`, `remotePath`, `file`.

### 8. `POST /api/v1/sftp/mkdir` & `POST /api/v1/sftp/delete`
Create remote directory or delete remote file/directory.
