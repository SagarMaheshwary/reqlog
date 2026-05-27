package app

import (
	"github.com/pkg/sftp"
	"github.com/sagarmaheshwary/reqlog/internal/scanner"
	"golang.org/x/crypto/ssh"
)

type scannerSource struct {
	scanner scanner.Scanner
	sources []string

	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func (s *scannerSource) Close() {
	if s.sftpClient != nil {
		_ = s.sftpClient.Close()
	}

	if s.sshClient != nil {
		_ = s.sshClient.Close()
	}
}
