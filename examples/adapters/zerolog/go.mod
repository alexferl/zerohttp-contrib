module github.com/alexferl/zerohttp-contrib/examples/adapters/zerolog

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.70.0
	github.com/alexferl/zerohttp-contrib/adapters/zerolog v0.2.0
	github.com/rs/zerolog v1.35.0
)

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

replace github.com/alexferl/zerohttp-contrib/adapters/zerolog => ../../../adapters/zerolog
