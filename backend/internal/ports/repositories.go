package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	ListAll(ctx context.Context) ([]domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type VaultRepository interface {
	Create(ctx context.Context, vault *domain.Vault) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Vault, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Vault, error)
	
	// Member management
	AddMember(ctx context.Context, member *domain.VaultMember) error
	GetMemberRole(ctx context.Context, vaultID, userID uuid.UUID) (domain.VaultRole, error)
	ListMembers(ctx context.Context, vaultID uuid.UUID) ([]domain.VaultMember, error)
	
	// Item management
	CreateItem(ctx context.Context, item *domain.VaultItem) error
	GetItemByID(ctx context.Context, vaultID, itemID uuid.UUID) (*domain.VaultItem, error)
	ListItemsByVault(ctx context.Context, vaultID uuid.UUID, kind *domain.ItemKind) ([]domain.VaultItem, error)
	UpdateItem(ctx context.Context, item *domain.VaultItem) error
	DeleteItem(ctx context.Context, vaultID, itemID uuid.UUID) error
	MoveItem(ctx context.Context, sourceVaultID, targetVaultID, itemID uuid.UUID) error
}

type SessionLogRepository interface {
	Create(ctx context.Context, log *domain.SessionLog) error
	ListByVault(ctx context.Context, vaultID uuid.UUID) ([]domain.SessionLog, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SessionLog, error)
}
