package ssh_reverse_tunnel

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	CanelledError = errors.New("Cancelled")
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

	clientLock *sync.Mutex

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

func NewClient(config ClientConfig) *Client {
	return &Client{
		Client: nil,
		config: config,
		done:   make(chan bool, 4),

		clientLock: &sync.Mutex{},
	}
}

func (c *Client) Connect() (err error) {
	check := func() bool {
		c.clientLock.Lock()
		defer c.clientLock.Unlock()
		if c.Client != nil {
			return true
		}
		return false
	}
	if check() {
		return nil
	}

	var sshConn *ssh.Client = nil
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}

		if err == nil {
			c.clientLock.Lock()
			defer c.clientLock.Unlock()
			c.Client = sshConn
		}
	}()
	ready := false
	for !ready {
		select {
		case <-c.done:
			break
		default:
			ready = true
		}
	}

	sshConn, err = sshConnection(c.config.SSHServer, &c.config.ClientConfig)
	if err != nil {
		return err
	}

	// sshConn.Listen()
	listener, err := sshConn.Listen("tcp", c.config.Remote.String())
	if err != nil {
		return err
	}

	go c.forward(listener)

	return nil
}

func (c *Client) duplexCopy(remoteConn, localConn net.Conn) (err error) {
	chDone := make(chan error, 2)
	defer close(chDone)

	// Start remote -> local data transfer
	go func(chDone chan<- error) {
		defer recover()
		_, err := io.Copy(remoteConn, localConn)
		chDone <- errors.New(fmt.Sprintf("Remote write error: %s", err.Error()))
	}(chDone)

	// Start local -> remote data transfer
	go func(chDone chan<- error) {
		defer recover()
		_, err := io.Copy(localConn, remoteConn)
		chDone <- errors.New(fmt.Sprintf("Local write error: %s", err.Error()))
	}(chDone)

	select {
	case a := <-c.done:
		c.done <- a
		err = CanelledError
		break
	case e := <-chDone:
		err = e
		break
	}
	return
}

func (c *Client) forward(listener net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
		}
		listener.Close()
	}()

	config := c.config
	for {
		select {
		default:
			err = func() error {
				localConn, err := net.DialTimeout(
					"tcp", config.Local.String(), config.Local.ConnectTimeout,
				)
				if err != nil {
					return err
				}
				defer localConn.Close()

				remoteConn, err := listener.Accept()
				if err != nil {
					return err
				}
				defer remoteConn.Close()

				return c.duplexCopy(remoteConn, localConn)
			}()

			if err != nil {
				log.Printf("Connect Error: %s", err.Error())
			}
		}
	}
}

func (c *Client) Close() error {
	full := false
	for !full {
		select {
		case c.done <- true:
			break
		default:
			full = true
		}
	}

	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}
