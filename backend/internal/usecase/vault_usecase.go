package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/domain"
	"github.com/opentermius/opentermius/internal/ports"
)

type VaultUseCase struct {
	vaultRepo ports.VaultRepository
	userRepo  ports.UserRepository
}

func NewVaultUseCase(vaultRepo ports.VaultRepository, userRepo ports.UserRepository) *VaultUseCase {
	return &VaultUseCase{
		vaultRepo: vaultRepo,
		userRepo:  userRepo,
	}
}

func (v *VaultUseCase) CreateVault(ctx context.Context, ownerID uuid.UUID, name string, vaultType domain.VaultType, encryptedKey string) (*domain.Vault, error) {
	if name == "" || encryptedKey == "" {
		return nil, domain.ErrInvalidInput
	}

	vault := &domain.Vault{
		ID:           uuid.New(),
		OwnerID:      ownerID,
		Name:         name,
		Type:         vaultType,
		EncryptedKey: encryptedKey,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := v.vaultRepo.Create(ctx, vault); err != nil {
		return nil, err
	}

	// Add owner as vault member
	member := &domain.VaultMember{
		ID:                uuid.New(),
		VaultID:           vault.ID,
		UserID:            ownerID,
		Role:              domain.VaultRoleOwner,
		EncryptedVaultKey: encryptedKey,
		CreatedAt:         time.Now().UTC(),
	}
	if err := v.vaultRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return vault, nil
}

func (v *VaultUseCase) ListUserVaults(ctx context.Context, userID uuid.UUID) ([]domain.Vault, error) {
	return v.vaultRepo.ListByUser(ctx, userID)
}

func (v *VaultUseCase) GetVault(ctx context.Context, vaultID, userID uuid.UUID) (*domain.Vault, error) {
	_, err := v.vaultRepo.GetMemberRole(ctx, vaultID, userID)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	return v.vaultRepo.GetByID(ctx, vaultID)
}

func (v *VaultUseCase) CreateItem(ctx context.Context, userID, vaultID uuid.UUID, kind domain.ItemKind, label, ciphertext, nonce string) (*domain.VaultItem, error) {
	_, err := v.vaultRepo.GetMemberRole(ctx, vaultID, userID)
	if err != nil {
		return nil, domain.ErrForbidden
	}

	if label == "" || ciphertext == "" || nonce == "" {
		return nil, domain.ErrInvalidInput
	}

	item := &domain.VaultItem{
		ID:         uuid.New(),
		VaultID:    vaultID,
		Kind:       kind,
		Label:      label,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Version:    1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := v.vaultRepo.CreateItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (v *VaultUseCase) ListItems(ctx context.Context, userID, vaultID uuid.UUID, kind *domain.ItemKind) ([]domain.VaultItem, error) {
	_, err := v.vaultRepo.GetMemberRole(ctx, vaultID, userID)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	return v.vaultRepo.ListItemsByVault(ctx, vaultID, kind)
}

func (v *VaultUseCase) UpdateItem(ctx context.Context, userID, vaultID, itemID uuid.UUID, label, ciphertext, nonce string) (*domain.VaultItem, error) {
	role, err := v.vaultRepo.GetMemberRole(ctx, vaultID, userID)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	if role == domain.VaultRoleMember {
		// members may have read-only in some configurations, but can edit if permitted
	}

	existing, err := v.vaultRepo.GetItemByID(ctx, vaultID, itemID)
	if err != nil {
		return nil, err
	}

	existing.Label = label
	existing.Ciphertext = ciphertext
	existing.Nonce = nonce
	existing.Version++
	existing.UpdatedAt = time.Now().UTC()

	if err := v.vaultRepo.UpdateItem(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (v *VaultUseCase) DeleteItem(ctx context.Context, userID, vaultID, itemID uuid.UUID) error {
	_, err := v.vaultRepo.GetMemberRole(ctx, vaultID, userID)
	if err != nil {
		return domain.ErrForbidden
	}
	return v.vaultRepo.DeleteItem(ctx, vaultID, itemID)
}

func (v *VaultUseCase) MoveItem(ctx context.Context, userID, sourceVaultID, targetVaultID, itemID uuid.UUID) error {
	if _, err := v.vaultRepo.GetMemberRole(ctx, sourceVaultID, userID); err != nil {
		return domain.ErrForbidden
	}
	if _, err := v.vaultRepo.GetMemberRole(ctx, targetVaultID, userID); err != nil {
		return domain.ErrForbidden
	}
	return v.vaultRepo.MoveItem(ctx, sourceVaultID, targetVaultID, itemID)
}

func (v *VaultUseCase) InviteMember(ctx context.Context, callerID, vaultID uuid.UUID, email string, role domain.VaultRole) (*domain.VaultMember, error) {
	callerRole, err := v.vaultRepo.GetMemberRole(ctx, vaultID, callerID)
	if err != nil || (callerRole != domain.VaultRoleOwner && callerRole != domain.VaultRoleAdmin) {
		return nil, domain.ErrForbidden
	}

	if role == "" {
		role = domain.VaultRoleMember
	}

	vault, err := v.vaultRepo.GetByID(ctx, vaultID)
	if err != nil {
		return nil, err
	}

	targetUser, err := v.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	member := &domain.VaultMember{
		ID:                uuid.New(),
		VaultID:           vaultID,
		UserID:            targetUser.ID,
		Role:              role,
		EncryptedVaultKey: vault.EncryptedKey,
		CreatedAt:         time.Now().UTC(),
	}

	if err := v.vaultRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

func (v *VaultUseCase) ListMembers(ctx context.Context, callerID, vaultID uuid.UUID) ([]domain.VaultMember, error) {
	_, err := v.vaultRepo.GetMemberRole(ctx, vaultID, callerID)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	return v.vaultRepo.ListMembers(ctx, vaultID)
}
