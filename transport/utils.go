package transport

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

func IsValidIP(addr string) bool {
	ip := net.ParseIP(addr)
	return ip.IsGlobalUnicast() && !ip.IsInterfaceLocalMulticast()
}

// ExtractHostPort from address
func ExtractHostPort(addr string) (host string, port uint64, err error) {
	var ports string
	host, ports, err = net.SplitHostPort(addr)
	if err != nil {
		return
	}
	port, err = strconv.ParseUint(ports, 10, 16) //nolint:mnd
	return
}

func ExtractPort(lis net.Listener) (int, bool) {
	if addr, ok := lis.Addr().(*net.TCPAddr); ok {
		return addr.Port, true
	}
	return 0, false
}

func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			ip := ipnet.IP.To4()
			if ip == nil {
				continue // 不是 IPv4
			}
			if !isValidBindableIPv4(ip) {
				continue
			}
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("未找到合适的本地 IPv4 地址")
}

func isValidBindableIPv4(ip net.IP) bool {
	// 回环 127.0.0.1
	if ip.IsLoopback() {
		return false
	}

	// 未指定 0.0.0.0
	if ip.IsUnspecified() {
		return false
	}

	// 多播地址 224.0.0.0 ~ 239.255.255.255
	if ip[0] >= 224 && ip[0] <= 239 {
		return false
	}

	// Link-local 169.254.x.x
	if ip[0] == 169 && ip[1] == 254 {
		return false
	}

	// 保留地址段 240.0.0.0+
	if ip[0] >= 240 {
		return false
	}

	// 示例地址
	if (ip[0] == 192 && ip[1] == 0 && ip[2] == 2) ||
		(ip[0] == 198 && ip[1] == 51 && ip[2] == 100) ||
		(ip[0] == 203 && ip[1] == 0 && ip[2] == 113) {
		return false
	}

	return true
}

func GetPublicIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func GetIPAddressByInterfaceName(ifName string) (string, error) {
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

func AdjustAddress(hostPort string, lis net.Listener) (string, error) {
	addr, port, err := net.SplitHostPort(hostPort)
	if err != nil && lis == nil {
		return "", err
	}
	if lis != nil {
		p, ok := ExtractPort(lis)
		if !ok {
			return "", fmt.Errorf("failed to extract port: %v", lis.Addr())
		}
		port = strconv.Itoa(p)
	}
	if len(addr) > 0 && (addr != "0.0.0.0" && addr != "[::]" && addr != "::") {
		return net.JoinHostPort(addr, port), nil
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	minIndex := int(^uint(0) >> 1)
	ips := make([]net.IP, 0)
	for _, iface := range ifaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}
		if iface.Index >= minIndex && len(ips) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for i, rawAddr := range addrs {
			var ip net.IP
			switch addr := rawAddr.(type) {
			case *net.IPAddr:
				ip = addr.IP
			case *net.IPNet:
				ip = addr.IP
			default:
				continue
			}
			if IsValidIP(ip.String()) {
				minIndex = iface.Index
				if i == 0 {
					ips = make([]net.IP, 0, 1)
				}
				ips = append(ips, ip)
				if ip.To4() != nil {
					break
				}
			}
		}
	}
	if len(ips) != 0 {
		return net.JoinHostPort(ips[len(ips)-1].String(), port), nil
	}
	return "", nil
}

func AdjustScheme(scheme string, isSecure bool) string {
	if isSecure {
		return scheme + "s"
	}
	return scheme
}

// NewRegistryEndpoint creates a new registry endpoint URL.
func NewRegistryEndpoint(serviceKindName string, host string) *url.URL {
	return &url.URL{Scheme: serviceKindName, Host: host}
}
