module github.com/alexferl/zerohttp-contrib/examples/extensions/autocert

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.80.0
	github.com/alexferl/zerohttp-contrib/extensions/autocert v0.2.0
	golang.org/x/crypto v0.49.0
)

require (
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace github.com/alexferl/zerohttp-contrib/extensions/autocert => ../../../extensions/autocert
