module github.com/alexferl/zerohttp-contrib/examples/middleware/compress

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.58.0
	github.com/alexferl/zerohttp-contrib/middleware/compress v0.2.0
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
)

replace github.com/alexferl/zerohttp-contrib/middleware/compress => ../../../middleware/compress
