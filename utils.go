package ssh_reverse_tunnel

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

var (
	PEMDecodeError = errors.New("Cannot Decode PEM.")
)

func OpenSSHAuthMethod(keyFile string) (ssh.AuthMethod, error) {
	key, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(key)
	if block == nil {
		return nil, PEMDecodeError
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// Create the Signer for this private key.
	signer, err := ssh.NewSignerFromKey(rsaKey)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}
