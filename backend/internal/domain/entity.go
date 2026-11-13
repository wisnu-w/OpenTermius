package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type VaultType string

const (
	VaultTypePersonal VaultType = "personal"
	VaultTypeTeam     VaultType = "team"
)

type VaultRole string

const (
	VaultRoleOwner  VaultRole = "owner"
	VaultRoleAdmin  VaultRole = "admin"
	VaultRoleMember VaultRole = "member"
)

type Vault struct {
	ID           uuid.UUID `json:"id"`
	OwnerID      uuid.UUID `json:"owner_id"`
	Name         string    `json:"name"`
	Type         VaultType `json:"type"`
	EncryptedKey string    `json:"encrypted_key"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type VaultMember struct {
	ID                uuid.UUID `json:"id"`
	VaultID           uuid.UUID `json:"vault_id"`
	UserID            uuid.UUID `json:"user_id"`
	Role              VaultRole `json:"role"`
	EncryptedVaultKey string    `json:"encrypted_vault_key"`
	CreatedAt         time.Time `json:"created_at"`
}

type ItemKind string

const (
	ItemKindHost     ItemKind = "host"
	ItemKindKey      ItemKind = "key"
	ItemKindIdentity ItemKind = "identity"
	ItemKindSnippet  ItemKind = "snippet"
	ItemKindTunnel   ItemKind = "tunnel"
)

type VaultItem struct {
	ID         uuid.UUID `json:"id"`
	VaultID    uuid.UUID `json:"vault_id"`
	Kind       ItemKind  `json:"kind"`
	Label      string    `json:"label"`
	Ciphertext string    `json:"ciphertext"`
	Nonce      string    `json:"nonce"`
	Version    int64     `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SessionLog struct {
	ID           uuid.UUID  `json:"id"`
	VaultID      uuid.UUID  `json:"vault_id"`
	UserID       uuid.UUID  `json:"user_id"`
	HostLabel    string     `json:"host_label"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	RecordedData string     `json:"recorded_data"`
}
