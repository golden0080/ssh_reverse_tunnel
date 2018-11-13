package ssh_reverse_tunnel

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

func OpenSSHPrivateKeyFile(keyFile string) (ssh.AuthMethod, error) {
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}
