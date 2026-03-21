module github.com/alexferl/zerohttp-contrib/examples/middleware/idempotency

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.49.0
	github.com/alexferl/zerohttp-contrib/middleware/idempotency v0.0.0
	github.com/redis/go-redis/v9 v9.18.0
)

replace (
	github.com/alexferl/zerohttp => ../../../../zerohttp
	github.com/alexferl/zerohttp-contrib/middleware/idempotency => ../../../middleware/idempotency
)
