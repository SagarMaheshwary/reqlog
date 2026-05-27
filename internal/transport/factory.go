package transport

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func NewFileSystem(client *sftp.Client) FileSystem {
	if client != nil {
		return &SSHFileSystem{client: client}
	}

	return &LocalFileSystem{}
}

func NewExecutor(client *ssh.Client) Executor {
	if client != nil {
		return &SSHExecutor{client: client}
	}

	return &LocalExecutor{}
}
