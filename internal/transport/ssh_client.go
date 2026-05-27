package transport

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"golang.org/x/crypto/ssh"
)

func NewSSHClient(host config.Host) (*ssh.Client, error) {
	key, err := os.ReadFile(host.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	addr := host.Host + ":" + strconv.Itoa(host.Port)
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect ssh client: %v", err)
	}

	return client, nil
}
