package transport

import (
	"net"
	"testing"
)

func TestGetIPAddressByInterfaceName(t *testing.T) {
	// 创建一个虚拟网络接口名称用于测试
	// 请根据实际环境替换为有效的网络接口名称
	interfaceName := "lo" // Linux下的回环接口名称，Windows可能需要替换为 "Loopback Pseudo-Interface 1"

	ip, err := GetIPAddressByInterfaceName(interfaceName)
	if err != nil {
		t.Fatalf("获取接口 %s 的IP地址失败: %v", interfaceName, err)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To4() == nil {
		t.Fatalf("获取的IP地址 %s 不是有效的IPv4地址", ip)
	}

	t.Logf("接口 %s 的IPv4地址: %s", interfaceName, ip)
}

func TestAdjustAddress(t *testing.T) {
	// 测试用例1: 提供hostPort和监听器
	hostPort := "127.0.0.1:8080"
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		t.Fatalf("无法创建监听器: %v", err)
	}
	defer listener.Close()

	result, err := AdjustAddress(hostPort, listener)
	if err != nil {
		t.Fatalf("AdjustAddress失败: %v", err)
	}
	t.Logf("原始地址 %s，实际地址 %s", hostPort, result)

	// 测试用例2: 仅提供hostPort
	hostPort = "0.0.0.0:8080"
	result, err = AdjustAddress(hostPort, nil)
	if err != nil {
		t.Fatalf("AdjustAddress失败: %v", err)
	}
	t.Logf("原始地址 %s，实际地址 %s", hostPort, result)

	// 测试用例3: 提供监听器但hostPort为空
	hostPort = ":7070"
	listener, err = net.Listen("tcp", hostPort)
	if err != nil {
		t.Fatalf("无法创建监听器: %v", err)
	}
	defer listener.Close()

	result, err = AdjustAddress("", listener)
	if err != nil {
		t.Fatalf("AdjustAddress失败: %v", err)
	}
	t.Logf("原始地址 %s，实际地址 %s", hostPort, result)
}
