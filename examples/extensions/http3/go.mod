module github.com/alexferl/zerohttp-contrib/examples/extensions/http3

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.86.0
	github.com/alexferl/zerohttp-contrib/extensions/http3 v0.2.0
)

require (
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
)

replace github.com/alexferl/zerohttp-contrib/extensions/http3 => ../../../extensions/http3
