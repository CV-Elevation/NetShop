module netshop/services/cart

go 1.26.1

replace kuoz/netshop/platform/shared/proto/cart => ../../shared/gen/cart

replace kuoz/netshop/platform/shared/proto/product => ../../shared/gen/product

replace kuoz/netshop/platform/shared/proto/common => ../../shared/gen/common

require (
	github.com/redis/go-redis/v9 v9.18.0
	google.golang.org/grpc v1.79.3
	kuoz/netshop/platform/shared/proto/cart v0.0.0-00010101000000-000000000000
	kuoz/netshop/platform/shared/proto/common v0.0.0-00010101000000-000000000000
	kuoz/netshop/platform/shared/proto/product v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
