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

func getIPByInterfaceName(ifName string) (string, error) {
	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("未找到接口 %s 的IPv4地址", ifName)
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//fmt.Println(ipnet.IP.String())
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("未找到非回环IPv4地址")
}

func getPublicIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func generatePort(min, max int) int {
	return rand.Intn(max-min) + min
}
