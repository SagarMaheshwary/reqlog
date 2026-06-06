package app

import (
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
	"golang.org/x/crypto/ssh"
)

type scannerSource struct {
	scanner scanner.Scanner
	sources []string

	sshClient *ssh.Client
}

func (s *scannerSource) Close() {
	if s.sshClient != nil {
		_ = s.sshClient.Close()
	}
}
