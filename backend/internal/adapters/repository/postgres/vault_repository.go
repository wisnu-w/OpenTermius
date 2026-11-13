package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opentermius/opentermius/internal/domain"
)

type VaultRepository struct {
	pool *pgxpool.Pool
}

func NewVaultRepository(pool *pgxpool.Pool) *VaultRepository {
	return &VaultRepository{pool: pool}
}

func (r *VaultRepository) Create(ctx context.Context, v *domain.Vault) error {
	query := `
		INSERT INTO vaults (id, owner_id, name, type, encrypted_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query, v.ID, v.OwnerID, v.Name, string(v.Type), v.EncryptedKey, v.CreatedAt, v.UpdatedAt)
	return err
}

func (r *VaultRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vault, error) {
	query := `
		SELECT id, owner_id, name, type, encrypted_key, created_at, updated_at
		FROM vaults
		WHERE id = $1
	`
	var v domain.Vault
	var vType string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.OwnerID, &v.Name, &vType, &v.EncryptedKey, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	v.Type = domain.VaultType(vType)
	return &v, nil
}

func (r *VaultRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Vault, error) {
	query := `
		SELECT v.id, v.owner_id, v.name, v.type, v.encrypted_key, v.created_at, v.updated_at
		FROM vaults v
		INNER JOIN vault_members vm ON vm.vault_id = v.id
		WHERE vm.user_id = $1
		ORDER BY v.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vaults []domain.Vault
	for rows.Next() {
		var v domain.Vault
		var vType string
		if err := rows.Scan(&v.ID, &v.OwnerID, &v.Name, &vType, &v.EncryptedKey, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		v.Type = domain.VaultType(vType)
		vaults = append(vaults, v)
	}
	return vaults, nil
}

func (r *VaultRepository) AddMember(ctx context.Context, m *domain.VaultMember) error {
	query := `
		INSERT INTO vault_members (id, vault_id, user_id, role, encrypted_vault_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (vault_id, user_id) DO UPDATE
		SET role = EXCLUDED.role, encrypted_vault_key = EXCLUDED.encrypted_vault_key
	`
	_, err := r.pool.Exec(ctx, query, m.ID, m.VaultID, m.UserID, string(m.Role), m.EncryptedVaultKey, m.CreatedAt)
	return err
}

func (r *VaultRepository) GetMemberRole(ctx context.Context, vaultID, userID uuid.UUID) (domain.VaultRole, error) {
	query := `
		SELECT role
		FROM vault_members
		WHERE vault_id = $1 AND user_id = $2
	`
	var role string
	err := r.pool.QueryRow(ctx, query, vaultID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrForbidden
		}
		return "", err
	}
	return domain.VaultRole(role), nil
}

func (r *VaultRepository) ListMembers(ctx context.Context, vaultID uuid.UUID) ([]domain.VaultMember, error) {
	query := `
		SELECT id, vault_id, user_id, role, encrypted_vault_key, created_at
		FROM vault_members
		WHERE vault_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, vaultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.VaultMember
	for rows.Next() {
		var m domain.VaultMember
		var role string
		if err := rows.Scan(&m.ID, &m.VaultID, &m.UserID, &role, &m.EncryptedVaultKey, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Role = domain.VaultRole(role)
		members = append(members, m)
	}
	return members, nil
}

func (r *VaultRepository) CreateItem(ctx context.Context, item *domain.VaultItem) error {
	query := `
		INSERT INTO vault_items (id, vault_id, kind, label, ciphertext, nonce, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		item.ID, item.VaultID, string(item.Kind), item.Label, item.Ciphertext, item.Nonce, item.Version, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *VaultRepository) GetItemByID(ctx context.Context, vaultID, itemID uuid.UUID) (*domain.VaultItem, error) {
	query := `
		SELECT id, vault_id, kind, label, ciphertext, nonce, version, created_at, updated_at
		FROM vault_items
		WHERE vault_id = $1 AND id = $2
	`
	var item domain.VaultItem
	var kind string
	err := r.pool.QueryRow(ctx, query, vaultID, itemID).Scan(
		&item.ID, &item.VaultID, &kind, &item.Label, &item.Ciphertext, &item.Nonce, &item.Version, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	item.Kind = domain.ItemKind(kind)
	return &item, nil
}

func (r *VaultRepository) ListItemsByVault(ctx context.Context, vaultID uuid.UUID, kind *domain.ItemKind) ([]domain.VaultItem, error) {
	var rows pgx.Rows
	var err error

	if kind != nil {
		query := `
			SELECT id, vault_id, kind, label, ciphertext, nonce, version, created_at, updated_at
			FROM vault_items
			WHERE vault_id = $1 AND kind = $2
			ORDER BY label ASC
		`
		rows, err = r.pool.Query(ctx, query, vaultID, string(*kind))
	} else {
		query := `
			SELECT id, vault_id, kind, label, ciphertext, nonce, version, created_at, updated_at
			FROM vault_items
			WHERE vault_id = $1
			ORDER BY label ASC
		`
		rows, err = r.pool.Query(ctx, query, vaultID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.VaultItem
	for rows.Next() {
		var item domain.VaultItem
		var itemKind string
		if err := rows.Scan(
			&item.ID, &item.VaultID, &itemKind, &item.Label, &item.Ciphertext, &item.Nonce, &item.Version, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Kind = domain.ItemKind(itemKind)
		items = append(items, item)
	}
	return items, nil
}

func (r *VaultRepository) UpdateItem(ctx context.Context, item *domain.VaultItem) error {
	query := `
		UPDATE vault_items
		SET label = $1, ciphertext = $2, nonce = $3, version = $4, updated_at = $5
		WHERE id = $6 AND vault_id = $7
	`
	_, err := r.pool.Exec(ctx, query,
		item.Label, item.Ciphertext, item.Nonce, item.Version, item.UpdatedAt, item.ID, item.VaultID,
	)
	return err
}

func (r *VaultRepository) DeleteItem(ctx context.Context, vaultID, itemID uuid.UUID) error {
	query := `
		DELETE FROM vault_items
		WHERE vault_id = $1 AND id = $2
	`
	_, err := r.pool.Exec(ctx, query, vaultID, itemID)
	return err
}

func (r *VaultRepository) MoveItem(ctx context.Context, sourceVaultID, targetVaultID, itemID uuid.UUID) error {
	query := `
		UPDATE vault_items
		SET vault_id = $1, updated_at = NOW()
		WHERE id = $2 AND vault_id = $3
	`
	_, err := r.pool.Exec(ctx, query, targetVaultID, itemID, sourceVaultID)
	return err
}
