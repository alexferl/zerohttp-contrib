module github.com/alexferl/zerohttp-contrib/examples/adapters/huma

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.58.0
	github.com/alexferl/zerohttp-contrib/adapters/huma v0.2.0
	github.com/danielgtaylor/huma/v2 v2.37.2
)

replace github.com/alexferl/zerohttp-contrib/adapters/huma => ../../../adapters/huma
