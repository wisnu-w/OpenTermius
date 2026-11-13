package ssh_test

import (
	"context"
	"testing"
	"time"

	"github.com/opentermius/opentermius/internal/adapters/ssh"
	"github.com/opentermius/opentermius/internal/ports"
)

func TestSSHService_UnreachableHost(t *testing.T) {
	service := ssh.NewSSHService()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try connecting to a non-existent port
	_, err := service.Connect(
		ctx,
		"127.0.0.1",
		19999, // Unused port
		"testuser",
		ports.SSHAuth{Type: "password", Password: "test"},
		80,
		24,
		nil,
	)

	if err == nil {
		t.Fatal("Expected connection error for unreachable host, got nil")
	}
}
