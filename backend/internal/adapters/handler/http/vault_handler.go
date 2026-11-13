package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/opentermius/opentermius/internal/domain"
	"github.com/opentermius/opentermius/internal/ports"
)

type VaultHandler struct {
	vaultService ports.VaultService
}

func NewVaultHandler(vaultService ports.VaultService) *VaultHandler {
	return &VaultHandler{vaultService: vaultService}
}

type createVaultRequest struct {
	Name         string           `json:"name"`
	Type         domain.VaultType `json:"type"`
	EncryptedKey string           `json:"encrypted_key"`
}

type createItemRequest struct {
	Kind       domain.ItemKind `json:"kind"`
	Label      string          `json:"label"`
	Ciphertext string          `json:"ciphertext"`
	Nonce      string          `json:"nonce"`
}

type updateItemRequest struct {
	Label      string `json:"label"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
}

func (h *VaultHandler) ListVaults(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaults, err := h.vaultService.ListUserVaults(r.Context(), uid)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list vaults")
		return
	}
	if vaults == nil {
		vaults = []domain.Vault{}
	}
	writeJSON(w, http.StatusOK, vaults)
}

func (h *VaultHandler) CreateVault(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	var req createVaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" {
		req.Type = domain.VaultTypePersonal
	}

	vault, err := h.vaultService.CreateVault(r.Context(), uid, req.Name, req.Type, req.EncryptedKey)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, "name and encrypted_key are required")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to create vault")
		return
	}

	writeJSON(w, http.StatusCreated, vault)
}

func (h *VaultHandler) GetVault(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}

	vault, err := h.vaultService.GetVault(r.Context(), vaultID, uid)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied to vault")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "vault not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch vault")
		return
	}

	writeJSON(w, http.StatusOK, vault)
}

func (h *VaultHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}

	var kindFilter *domain.ItemKind
	if k := r.URL.Query().Get("kind"); k != "" {
		kind := domain.ItemKind(k)
		kindFilter = &kind
	}

	items, err := h.vaultService.ListItems(r.Context(), uid, vaultID, kindFilter)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied to vault")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to list items")
		return
	}
	if items == nil {
		items = []domain.VaultItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *VaultHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.vaultService.CreateItem(r.Context(), uid, vaultID, req.Kind, req.Label, req.Ciphertext, req.Nonce)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied to vault")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, "label, ciphertext and nonce are required")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to create vault item")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *VaultHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var req updateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.vaultService.UpdateItem(r.Context(), uid, vaultID, itemID, req.Label, req.Ciphertext, req.Nonce)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "item not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update item")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *VaultHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	if err := h.vaultService.DeleteItem(r.Context(), uid, vaultID, itemID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type moveItemRequest struct {
	TargetVaultID string `json:"target_vault_id"`
}

func (h *VaultHandler) MoveItem(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	sourceVaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid source vault id")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var req moveItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetVaultID == "" {
		writeJSONError(w, http.StatusBadRequest, "valid target_vault_id is required")
		return
	}
	targetVaultID, err := uuid.Parse(req.TargetVaultID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid target vault id format")
		return
	}

	if err := h.vaultService.MoveItem(r.Context(), uid, sourceVaultID, targetVaultID, itemID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied to source or target vault")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to move item")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "moved"})
}

type inviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *VaultHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "valid email is required")
		return
	}

	member, err := h.vaultService.InviteMember(r.Context(), uid, vaultID, req.Email, domain.VaultRole(req.Role))
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "only vault owner or admin can invite members")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "user with this email is not registered yet")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to invite member")
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

func (h *VaultHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	uid := getUserID(r)
	vaultID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid vault id")
		return
	}

	members, err := h.vaultService.ListMembers(r.Context(), uid, vaultID)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSONError(w, http.StatusForbidden, "access denied")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	if members == nil {
		members = []domain.VaultMember{}
	}

	writeJSON(w, http.StatusOK, members)
}
