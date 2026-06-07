package transport

import (
	"golang.org/x/crypto/ssh"
)

func NewExecutor(client *ssh.Client) Executor {
	if client != nil {
		return &SSHExecutor{client: client}
	}

	return &LocalExecutor{}
}

func NewLogFileReader(executor Executor) LogFileReader {
	if executor != nil {
		return &SSHLogFileReader{executor: executor}
	}

	return &LocalLogFileReader{}
}
