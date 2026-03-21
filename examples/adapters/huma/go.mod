module github.com/alexferl/zerohttp-contrib/examples/adapters/huma

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.49.0
	github.com/alexferl/zerohttp-contrib/adapters/huma v0.0.0
	github.com/danielgtaylor/huma/v2 v2.32.0
)

replace (
	github.com/alexferl/zerohttp => ../../../../zerohttp
	github.com/alexferl/zerohttp-contrib/adapters/huma => ../../../adapters/huma
)
