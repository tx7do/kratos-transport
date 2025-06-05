package keepalive

import "os"

const (
	EnvKeyHost      = "KRATOS_TRANSPORT_KEEPALIVE_HOST"
	EnvKeyInterface = "KRATOS_TRANSPORT_KEEPALIVE_INTERFACE"
)

// SetKeepAliveHost sets the host for the keep-alive service.
func SetKeepAliveHost(host string) {
	_ = os.Setenv(EnvKeyHost, host)
}

// SetKeepAliveInterface sets the network interface for the keep-alive service.
func SetKeepAliveInterface(itf string) {
	_ = os.Setenv(EnvKeyInterface, itf)
}
