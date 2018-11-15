package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golden0080/ssh_reverse_tunnel"
	"golang.org/x/crypto/ssh"
)

var (
	SSHUser        = flag.String("ssh_user", "", "The SSH server login user.")
	SSHServer      = flag.String("ssh", "", "The remote ssh server.")
	SSHKey         = flag.String("ssh_key", "./.ssh/id_rsa", "The ssh login private key file.")
	RemoteHostPort = flag.String("remote", "", "The remote binding host:port.")
	LocalPort      = flag.Int(
		"local", 22, "The local port to bind for reverse tunnel.",
	)
	SSHTimeout = flag.Int64("ssh_timeout", 2000, "The SSH connect timeout, in millisecond")
)

func main() {
	flag.Parse()

	var serverHost string
	serverHost, serverPort, err := net.SplitHostPort(*SSHServer)
	if err != nil {
		serverHost = *SSHServer
	}
	var serverPortInt int
	if len(serverPort) > 0 {
		serverPortInt, err = strconv.Atoi(serverPort)
		if err != nil {
			log.Printf("Invalid SSH Server Port [%s]", serverPort)
		}
	} else {
		serverPortInt = 22
	}

	remoteHost, remotePort, err := net.SplitHostPort(*RemoteHostPort)
	if err != nil || len(remoteHost) == 0 || len(remotePort) == 0 {
		log.Printf("Invalid Remote Config [%s]", *RemoteHostPort)
		return
	}
	var remotePortInt int
	remotePortInt, err = strconv.Atoi(remotePort)
	if err != nil {
		log.Printf("Invalid Remote Port [%s]", remotePort)
	}

	keyLogin, err := ssh_reverse_tunnel.OpenSSHAuthMethod(*SSHKey)
	if err != nil {
		log.Printf("Unable to use key file[%s] for ssh: %s", *SSHKey, err.Error())
		return
	}

	sshConfig := &ssh.ClientConfig{
		User:    *SSHUser,
		Auth:    []ssh.AuthMethod{keyLogin},
		Timeout: time.Millisecond * time.Duration(*SSHTimeout),

		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	tunnelConfig := &ssh_reverse_tunnel.ClientConfig{
		ClientConfig: *sshConfig,
		SSHServer: ssh_reverse_tunnel.Endpoint{
			Host: serverHost,
			Port: serverPortInt,
		},
		Remote: ssh_reverse_tunnel.Endpoint{
			Host: remoteHost,
			Port: remotePortInt,
		},
		Local: ssh_reverse_tunnel.Endpoint{
			Host: "localhost",
			Port: *LocalPort,
		},
	}

	client := ssh_reverse_tunnel.NewClient(*tunnelConfig)
	err = client.Connect()

	if err != nil {
		log.Printf("Failed to connect: %s", err.Error())
		return
	}

	defer client.Close()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Printf("Exiting...\n")
}
