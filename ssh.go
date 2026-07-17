package main

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	host      string
	port      int
	username  string
	client    *ssh.Client
	connected bool
}

func (s *SSHClient) Connect(host string, port int, username, password string) error {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	s.host = host
	s.port = port
	s.username = username
	s.client = client
	s.connected = true
	return nil
}

func (s *SSHClient) Exec(command string) (string, error) {
	if !s.connected || s.client == nil {
		return "", fmt.Errorf("not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		s.connected = false
		return "", fmt.Errorf("session error: %w", err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 40, 120, modes); err != nil {
		return "", fmt.Errorf("pty error: %w", err)
	}

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)

	output := stdout.String()
	if s := stderr.String(); s != "" {
		if output != "" {
			output += "\n"
		}
		output += s
	}

	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			output += fmt.Sprintf("\n[exit code: %d]", exitErr.ExitStatus())
		} else {
			return output, fmt.Errorf("exec error: %w", err)
		}
	}

	return output, nil
}

func (s *SSHClient) Disconnect() {
	if s.client != nil {
		s.client.Close()
	}
	s.connected = false
	s.client = nil
}

// IsAlive checks if the SSH connection is still usable
func (s *SSHClient) IsAlive() bool {
	if s.client == nil {
		return false
	}
	// Try to create a new session — if it fails, connection is dead
	session, err := s.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}
