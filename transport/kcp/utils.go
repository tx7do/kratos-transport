package kcp

import (
	"crypto/sha1"

	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
)

const (
	DefaultBlockCryptPassword = "kratos-transport-kcp-password"
	DefaultBlockCryptSalt     = "kratos-transport-kcp-salt"
)

// NewBlockCryptFromPassword creates a new BlockCrypt using the given password and salt.
func NewBlockCryptFromPassword(password, salt string) kcp.BlockCrypt {
	key := pbkdf2.Key([]byte(password), []byte(salt), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	return block
}
