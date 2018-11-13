package ssh_reverse_tunnel

import (
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// Endpoint is a network location, with host and port.
type Endpoint struct {
	Host string
	Port int

	ConnectTimeout time.Duration
}

func (endpoint Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

// ClientConfig will confgure SSH connection and reserve tunnel.
type ClientConfig struct {
	ssh.ClientConfig

	SSHServer Endpoint

	Remote Endpoint
	Local  Endpoint
}

type Client struct {
	*ssh.Client

	config ClientConfig
	done   chan bool
}

func sshConnection(
	server Endpoint, config *ssh.ClientConfig,
) (*ssh.Client, error) {
	serverConn, err := ssh.Dial("tcp", server.String(), config)
	return serverConn, err
}

func tcpConnection(tcpEndpoint Endpoint) (*net.Conn, error) {
	conn, err := net.DialTimeout(
		"tcp", tcpEndpoint.String(), tcpEndpoint.ConnectTimeout,
	)
	return &conn, err
}

func NewClient(config *ClientConfig) *Client {
	return &Client{
		Client: nil,
		config: *config,
		done:   nil,
	}
}

func (c *Client) Connect() (err error) {
	if c.Client != nil {
		return nil
	}

	var sshConn *ssh.Client = nil
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}

		if err == nil {
			c.Client = sshConn
		}
	}()

	sshConn, err = sshConnection(c.config.SSHServer, &c.config.ClientConfig)
	if err != nil {
		return err
	}

	// sshConn.Listen()
	return nil
}
