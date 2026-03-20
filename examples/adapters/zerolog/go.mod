module github.com/alexferl/zerohttp-contrib/examples/adapters/zerolog

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.49.0
	github.com/alexferl/zerohttp-contrib/adapters/zerolog v0.0.0
	github.com/rs/zerolog v1.33.0
)

replace (
	github.com/alexferl/zerohttp => ../../../../zerohttp
	github.com/alexferl/zerohttp-contrib/adapters/zerolog => ../../../adapters/zerolog
)
