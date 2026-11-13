package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/opentermius/opentermius/internal/ports"
	"golang.org/x/crypto/ssh"
)

type sshSessionImpl struct {
	client    *ssh.Client
	session   *ssh.Session
	stdinPipe io.WriteCloser
	mu        sync.Mutex
	closed    bool
}

func (s *sshSessionImpl) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.stdinPipe == nil {
		return 0, io.EOF
	}
	return s.stdinPipe.Write(p)
}

func (s *sshSessionImpl) Resize(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.session == nil {
		return io.EOF
	}
	return s.session.WindowChange(rows, cols)
}

func (s *sshSessionImpl) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true

	if s.stdinPipe != nil {
		_ = s.stdinPipe.Close()
	}
	if s.session != nil {
		_ = s.session.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
	return nil
}

type SSHService struct{}

func NewSSHService() *SSHService {
	return &SSHService{}
}

func DialSSH(ctx context.Context, host string, port int, user string, auth ports.SSHAuth) (*ssh.Client, error) {
	if port <= 0 {
		port = 22
	}

	var authMethods []ssh.AuthMethod

	if auth.Type == "key" || auth.PrivateKey != "" {
		var signer ssh.Signer
		var err error
		if auth.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(auth.PrivateKey), []byte(auth.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(auth.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else if auth.Password != "" {
		authMethods = append(authMethods, ssh.Password(auth.Password))
	} else {
		// Keyboard-interactive / empty password fallback
		authMethods = append(authMethods, ssh.Password(""))
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // OpenTermius MVP
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// Dial with context-bounded timeout
	dialer := net.Dialer{Timeout: 10 * time.Second}
	netConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("network connection to %s failed: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
	if err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("SSH handshake failed with %s: %w", addr, err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (s *SSHService) Connect(
	ctx context.Context,
	host string,
	port int,
	user string,
	auth ports.SSHAuth,
	cols, rows int,
	stdout io.Writer,
) (ports.SSHSession, error) {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	client, err := DialSSH(ctx, host, port, user, auth)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, fmt.Errorf("failed to request pseudo-terminal (PTY): %w", err)
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	session.Stdout = stdout
	session.Stderr = stdout

	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, fmt.Errorf("failed to spawn remote shell: %w", err)
	}

	return &sshSessionImpl{
		client:    client,
		session:   session,
		stdinPipe: stdinPipe,
	}, nil
}
