module github.com/alexferl/zerohttp-contrib/examples/extensions/http3_autocert

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.70.0
	github.com/alexferl/zerohttp-contrib/extensions/autocert v0.2.0
	github.com/alexferl/zerohttp-contrib/extensions/http3 v0.2.0
	golang.org/x/crypto v0.49.0
)

require (
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace (
	github.com/alexferl/zerohttp-contrib/extensions/autocert => ../../../extensions/autocert
	github.com/alexferl/zerohttp-contrib/extensions/http3 => ../../../extensions/http3
)
