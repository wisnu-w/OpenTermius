package ports

import (
	"context"
	"io"
)

type SSHAuth struct {
	Type       string // "password" or "key"
	Password   string
	PrivateKey string
	Passphrase string
}

type SSHSession interface {
	io.Writer
	Resize(cols, rows int) error
	Close() error
}

type SSHService interface {
	Connect(ctx context.Context, host string, port int, user string, auth SSHAuth, cols, rows int, stdout io.Writer) (SSHSession, error)
}

type SFTPFileItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDir       bool   `json:"isDir"`
	Size        int64  `json:"size"`
	ModTime     string `json:"modTime"`
	Permissions string `json:"permissions"`
}

type SFTPService interface {
	List(ctx context.Context, host string, port int, user string, auth SSHAuth, remotePath string) ([]SFTPFileItem, string, error)
	Download(ctx context.Context, host string, port int, user string, auth SSHAuth, remotePath string) (io.ReadCloser, int64, string, error)
	Upload(ctx context.Context, host string, port int, user string, auth SSHAuth, remotePath string, r io.Reader) error
	CreateDir(ctx context.Context, host string, port int, user string, auth SSHAuth, remotePath string) error
	Delete(ctx context.Context, host string, port int, user string, auth SSHAuth, remotePath string) error
}
