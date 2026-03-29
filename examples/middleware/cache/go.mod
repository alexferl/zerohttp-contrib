module github.com/alexferl/zerohttp-contrib/examples/middleware/cache

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.70.0
	github.com/alexferl/zerohttp-contrib/middleware/cache v0.2.0
	github.com/redis/go-redis/v9 v9.18.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/atomic v1.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/alexferl/zerohttp-contrib/middleware/cache => ../../../middleware/cache
