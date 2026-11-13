package ssh

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/opentermius/opentermius/internal/ports"
	"github.com/pkg/sftp"
)

type SFTPService struct{}

func NewSFTPService() *SFTPService {
	return &SFTPService{}
}

func (s *SFTPService) List(
	ctx context.Context,
	host string,
	port int,
	user string,
	auth ports.SSHAuth,
	remotePath string,
) ([]ports.SFTPFileItem, string, error) {
	sshClient, err := DialSSH(ctx, host, port, user, auth)
	if err != nil {
		return nil, "", err
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize SFTP client: %w", err)
	}
	defer sftpClient.Close()

	targetPath := strings.TrimSpace(remotePath)
	if targetPath == "" || targetPath == "." {
		cwd, err := sftpClient.Getwd()
		if err == nil && cwd != "" {
			targetPath = cwd
		} else {
			targetPath = "/"
		}
	}

	realTarget, err := sftpClient.RealPath(targetPath)
	if err == nil && realTarget != "" {
		targetPath = realTarget
	}

	entries, err := sftpClient.ReadDir(targetPath)
	if err != nil {
		return nil, targetPath, fmt.Errorf("failed to read directory %s: %w", targetPath, err)
	}

	items := make([]ports.SFTPFileItem, 0, len(entries))
	for _, entry := range entries {
		itemPath := path.Join(targetPath, entry.Name())
		items = append(items, ports.SFTPFileItem{
			Name:        entry.Name(),
			Path:        itemPath,
			IsDir:       entry.IsDir(),
			Size:        entry.Size(),
			ModTime:     entry.ModTime().Format("2006-01-02 15:04:05"),
			Permissions: entry.Mode().String(),
		})
	}

	// Sort directories first, then alphabetical
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items, targetPath, nil
}

type sftpReadCloser struct {
	file       *sftp.File
	sftpClient *sftp.Client
	sshClient  io.Closer
}

func (r *sftpReadCloser) Read(p []byte) (int, error) {
	return r.file.Read(p)
}

func (r *sftpReadCloser) Close() error {
	_ = r.file.Close()
	_ = r.sftpClient.Close()
	_ = r.sshClient.Close()
	return nil
}

func (s *SFTPService) Download(
	ctx context.Context,
	host string,
	port int,
	user string,
	auth ports.SSHAuth,
	remotePath string,
) (io.ReadCloser, int64, string, error) {
	sshClient, err := DialSSH(ctx, host, port, user, auth)
	if err != nil {
		return nil, 0, "", err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, 0, "", fmt.Errorf("failed to initialize SFTP client: %w", err)
	}

	file, err := sftpClient.Open(remotePath)
	if err != nil {
		_ = sftpClient.Close()
		_ = sshClient.Close()
		return nil, 0, "", fmt.Errorf("failed to open remote file %s: %w", remotePath, err)
	}

	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		_ = sftpClient.Close()
		_ = sshClient.Close()
		return nil, 0, "", fmt.Errorf("failed to stat file: %w", err)
	}

	filename := path.Base(remotePath)
	return &sftpReadCloser{
		file:       file,
		sftpClient: sftpClient,
		sshClient:  sshClient,
	}, stat.Size(), filename, nil
}

func (s *SFTPService) Upload(
	ctx context.Context,
	host string,
	port int,
	user string,
	auth ports.SSHAuth,
	remotePath string,
	r io.Reader,
) error {
	sshClient, err := DialSSH(ctx, host, port, user, auth)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("failed to initialize SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Ensure parent dir exists
	dir := path.Dir(remotePath)
	_ = sftpClient.MkdirAll(dir)

	file, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", remotePath, err)
	}

	return nil
}

func (s *SFTPService) CreateDir(
	ctx context.Context,
	host string,
	port int,
	user string,
	auth ports.SSHAuth,
	remotePath string,
) error {
	sshClient, err := DialSSH(ctx, host, port, user, auth)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("failed to initialize SFTP client: %w", err)
	}
	defer sftpClient.Close()

	if err := sftpClient.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", remotePath, err)
	}
	return nil
}

func (s *SFTPService) Delete(
	ctx context.Context,
	host string,
	port int,
	user string,
	auth ports.SSHAuth,
	remotePath string,
) error {
	sshClient, err := DialSSH(ctx, host, port, user, auth)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("failed to initialize SFTP client: %w", err)
	}
	defer sftpClient.Close()

	stat, err := sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("path not found: %w", err)
	}

	if stat.IsDir() {
		if err := sftpClient.RemoveDirectory(remotePath); err != nil {
			return fmt.Errorf("failed to delete directory %s: %w", remotePath, err)
		}
	} else {
		if err := sftpClient.Remove(remotePath); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", remotePath, err)
		}
	}

	return nil
}
