package kcp

import (
	"crypto/sha1"
	"net"
	"net/url"

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

// AddrToURL net.Addr 转换为 url.URL
// 支持 tcp/udp
func AddrToURL(addr net.Addr) (*url.URL, error) {
	if addr == nil {
		return nil, nil
	}

	// 获取网络类型：tcp / udp
	network := addr.Network()
	// 获取地址：ip:port
	address := addr.String()

	// 拼接成 URL 字符串
	// 例如：tcp://127.0.0.1:8080
	uStr := network + "://" + address

	// 解析成 url.URL
	u, err := url.Parse(uStr)
	if err != nil {
		return nil, err
	}

	return u, nil
}
