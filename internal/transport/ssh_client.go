package transport

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"golang.org/x/crypto/ssh"
)

type SSHDialer func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)

func NewSSHClient(host config.Host, dial SSHDialer) (*ssh.Client, error) {
	key, err := os.ReadFile(host.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	addr := host.Host + ":" + strconv.Itoa(host.Port)
	client, err := dial("tcp", addr, &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         host.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect ssh client: %w", err)
	}

	return client, nil
}
