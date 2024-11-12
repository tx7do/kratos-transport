package keepalive

import (
	"fmt"
	"math/rand"
	"net"
)

func getIPAddress(interfaceName string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iFace := range interfaces {
		if iFace.Name == interfaceName {
			addrs, err := iFace.Addrs()
			if err != nil {
				return "", err
			}

			// 获取第一个IPv4地址
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					return ipNet.IP.String(), nil
				}
			}
			return "", fmt.Errorf("no IPv4 address found for interface %s", interfaceName)
		}
	}

	return "", fmt.Errorf("interface %s not found", interfaceName)
}

func generatePort(min, max int) int {
	return rand.Intn(max-min) + min
}
