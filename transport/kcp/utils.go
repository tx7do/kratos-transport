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
// 未显式配置 password/salt 时回退到内置默认值：空口令派生的密钥任何人都能推导，
// 与“不加密”无异；生产环境应始终通过 WithBlockCryptPassword/WithBlockCryptSalt 覆盖。
func NewBlockCryptFromPassword(password, salt string) kcp.BlockCrypt {
	if password == "" {
		LogWarn("block crypt password is empty, fallback to built-in default; set it via options in production")
		password = DefaultBlockCryptPassword
	}
	if salt == "" {
		salt = DefaultBlockCryptSalt
	}
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
