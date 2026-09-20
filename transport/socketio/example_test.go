package socketio_test

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	socketIo "github.com/googollee/go-socket.io"

	"github.com/tx7do/kratos-transport/transport/socketio"
)

// ExampleNewServer 演示启动一个 socket.io 事件服务：
// 注册连接、断开与自定义事件处理器，收到 notice 事件后回推 reply。
// 没有 Output 注释，go test 只编译不执行；实际运行需要 socket.io 客户端连接 :8800/socket.io/。
func ExampleNewServer() {
	ioSrv := socketio.NewServer(
		socketio.WithAddress(":8800"),
		socketio.WithCodec("json"),
		socketio.WithPath("/socket.io/"),
	)

	ioSrv.RegisterConnectHandler("/", func(conn socketIo.Conn) error {
		conn.SetContext("")
		log.Info("connected:", conn.ID())
		return nil
	})

	ioSrv.RegisterEventHandler("/", "notice", func(conn socketIo.Conn, msg string) {
		log.Info("notice:", msg)
		conn.Emit("reply", "have "+msg)
	})

	ioSrv.RegisterDisconnectHandler("/", func(conn socketIo.Conn, reason string) {
		log.Info("closed:", reason)
	})

	app := kratos.New(
		kratos.Name("socketio"),
		kratos.Server(
			ioSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
