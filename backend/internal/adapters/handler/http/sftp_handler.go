package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/opentermius/opentermius/internal/ports"
)

type SFTPHandler struct {
	sftpService ports.SFTPService
}

func NewSFTPHandler(sftpService ports.SFTPService) *SFTPHandler {
	return &SFTPHandler{sftpService: sftpService}
}

type sftpBaseReq struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthType   string `json:"authType"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	Passphrase string `json:"passphrase"`
	Path       string `json:"path"`
}

func (h *SFTPHandler) List(w http.ResponseWriter, r *http.Request) {
	var req sftpBaseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	auth := ports.SSHAuth{
		Type:       req.AuthType,
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
	}

	items, currentPath, err := h.sftpService.List(r.Context(), req.Host, req.Port, req.Username, auth, req.Path)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":  currentPath,
		"items": items,
	})
}

func (h *SFTPHandler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req sftpBaseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	auth := ports.SSHAuth{
		Type:       req.AuthType,
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
	}

	if err := h.sftpService.CreateDir(r.Context(), req.Host, req.Port, req.Username, auth, req.Path); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *SFTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req sftpBaseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	auth := ports.SSHAuth{
		Type:       req.AuthType,
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
	}

	if err := h.sftpService.Delete(r.Context(), req.Host, req.Port, req.Username, auth, req.Path); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *SFTPHandler) Download(w http.ResponseWriter, r *http.Request) {
	var req sftpBaseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	auth := ports.SSHAuth{
		Type:       req.AuthType,
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
	}

	rc, size, filename, err := h.sftpService.Download(r.Context(), req.Host, req.Port, req.Username, auth, req.Path)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))

	_, _ = io.Copy(w, rc)
}

func (h *SFTPHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form up to 100MB in memory buffer
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse multipart upload: "+err.Error())
		return
	}

	host := r.FormValue("host")
	portStr := r.FormValue("port")
	port, _ := strconv.Atoi(portStr)
	username := r.FormValue("username")
	authType := r.FormValue("authType")
	password := r.FormValue("password")
	privateKey := r.FormValue("privateKey")
	passphrase := r.FormValue("passphrase")
	remotePath := r.FormValue("remotePath")

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing upload file")
		return
	}
	defer file.Close()

	if strings.HasSuffix(remotePath, "/") || remotePath == "" {
		remotePath = strings.TrimSuffix(remotePath, "/") + "/" + header.Filename
	}

	auth := ports.SSHAuth{
		Type:       authType,
		Password:   password,
		PrivateKey: privateKey,
		Passphrase: passphrase,
	}

	if err := h.sftpService.Upload(r.Context(), host, port, username, auth, remotePath, file); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"filename": header.Filename,
		"path":     remotePath,
		"size":     header.Size,
	})
}
