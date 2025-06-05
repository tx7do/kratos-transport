package keepalive

import (
	"math/rand"
	"os"

	"github.com/tx7do/kratos-transport/transport"
)

func generatePort() int {
	return generateNumber(10000, 65535)
}

func generateNumber(minNum, maxNum int) int {
	return rand.Intn(maxNum-minNum) + minNum
}

func generateHost() (string, error) {
	if itf, ok := os.LookupEnv(EnvKeyInterface); ok {
		h, err := transport.GetIPAddressByInterfaceName(itf)
		if err != nil {
			return "", err
		}
		return h, nil
	} else if h, ok := os.LookupEnv(EnvKeyHost); ok {
		return h, nil
	} else {
		h, err := transport.GetLocalIP()
		if err != nil {
			return "", err
		}
		return h, nil
	}
}
