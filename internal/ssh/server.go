package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/git"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	cfg    *config.Config
	sshCfg *ssh.ServerConfig
	wg     sync.WaitGroup
}

func NewServer(cfg *config.Config) (*Server, error) {
	hostKey, err := loadOrGenerateHostKey(cfg.HostKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load host key: %w", err)
	}

	sshCfg := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	sshCfg.AddHostKey(hostKey)

	return &Server{cfg: cfg, sshCfg: sshCfg}, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	fmt.Printf("SSH server listening on port %d\n", s.cfg.Port)
	fmt.Printf("Repositories: %s\n", s.cfg.ReposDir)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Channel to signal shutdown
	done := make(chan struct{})

	go func() {
		<-sigChan
		fmt.Println("\nShutting down server...")
		listener.Close()
		close(done)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-done:
				// Graceful shutdown - wait for active connections
				s.wg.Wait()
				return nil
			default:
				fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.sshCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to handshake: %v\n", err)
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept channel: %v\n", err)
			continue
		}

		go s.handleSession(channel, requests)
	}
}

func (s *Server) handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		if req.Type == "exec" {
			// SSH protocol: first 4 bytes are length prefix
			if len(req.Payload) < 4 {
				fmt.Fprintf(channel.Stderr(), "Error: invalid SSH payload\n")
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
				return
			}

			cmdStr := string(req.Payload[4:])
			req.Reply(true, nil)

			cmd, err := git.ParseCommand(cmdStr)
			if err != nil {
				fmt.Fprintf(channel.Stderr(), "Error: %v\n", err)
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
				return
			}

			if err := cmd.Execute(s.cfg.ReposDir, channel, channel, channel.Stderr()); err != nil {
				fmt.Fprintf(channel.Stderr(), "Error: %v\n", err)
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
				return
			}

			channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
			return
		}
		req.Reply(req.Type == "shell", nil)
	}
}

func loadOrGenerateHostKey(path string) (ssh.Signer, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return generateHostKey(path)
	}

	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(keyData)
}

func generateHostKey(path string) (ssh.Signer, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	pemData := pem.EncodeToMemory(privateKeyBytes)
	if err := os.WriteFile(path, pemData, 0600); err != nil {
		return nil, err
	}

	return ssh.NewSignerFromKey(privateKey)
}
