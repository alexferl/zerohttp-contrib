module github.com/alexferl/zerohttp-contrib/examples/extensions/websocket

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.46.0
	github.com/alexferl/zerohttp-contrib/extensions/websocket v0.0.0
	github.com/gorilla/websocket v1.5.3
)

replace github.com/alexferl/zerohttp-contrib/extensions/websocket => ../../../extensions/websocket
