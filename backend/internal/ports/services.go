package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/domain"
)

type AuthService interface {
	Register(ctx context.Context, email, password, name string) (*domain.User, string, error)
	Login(ctx context.Context, email, password string) (*domain.User, string, error)
	ValidateToken(ctx context.Context, tokenString string) (uuid.UUID, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

type VaultService interface {
	CreateVault(ctx context.Context, ownerID uuid.UUID, name string, vaultType domain.VaultType, encryptedKey string) (*domain.Vault, error)
	ListUserVaults(ctx context.Context, userID uuid.UUID) ([]domain.Vault, error)
	GetVault(ctx context.Context, vaultID, userID uuid.UUID) (*domain.Vault, error)
	
	// Items
	CreateItem(ctx context.Context, userID, vaultID uuid.UUID, kind domain.ItemKind, label, ciphertext, nonce string) (*domain.VaultItem, error)
	ListItems(ctx context.Context, userID, vaultID uuid.UUID, kind *domain.ItemKind) ([]domain.VaultItem, error)
	UpdateItem(ctx context.Context, userID, vaultID, itemID uuid.UUID, label, ciphertext, nonce string) (*domain.VaultItem, error)
	DeleteItem(ctx context.Context, userID, vaultID, itemID uuid.UUID) error
	MoveItem(ctx context.Context, userID, sourceVaultID, targetVaultID, itemID uuid.UUID) error

	// Members
	InviteMember(ctx context.Context, callerID, vaultID uuid.UUID, email string, role domain.VaultRole) (*domain.VaultMember, error)
	ListMembers(ctx context.Context, callerID, vaultID uuid.UUID) ([]domain.VaultMember, error)
}

type SessionLogService interface {
	RecordLog(ctx context.Context, userID, vaultID uuid.UUID, hostLabel, data string) (*domain.SessionLog, error)
	ListLogs(ctx context.Context, userID, vaultID uuid.UUID) ([]domain.SessionLog, error)
}
