module netshop/services/frontend

go 1.26.1

require (
	github.com/golang-jwt/jwt/v5 v5.2.2
	google.golang.org/grpc v1.79.3
	kuoz/netshop/platform/shared/proto/ad v0.0.0
	kuoz/netshop/platform/shared/proto/cart v0.0.0
	kuoz/netshop/platform/shared/proto/common v0.0.0
	kuoz/netshop/platform/shared/proto/email v0.0.0
	kuoz/netshop/platform/shared/proto/product v0.0.0
	kuoz/netshop/platform/shared/proto/recommend v0.0.0
	kuoz/netshop/platform/shared/proto/user v0.0.0
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace kuoz/netshop/platform/shared/proto/user => ../../shared/gen/user

replace kuoz/netshop/platform/shared/proto/email => ../../shared/gen/email

replace kuoz/netshop/platform/shared/proto/product => ../../shared/gen/product

replace kuoz/netshop/platform/shared/proto/ad => ../../shared/gen/ad

replace kuoz/netshop/platform/shared/proto/recommend => ../../shared/gen/recommend

replace kuoz/netshop/platform/shared/proto/cart => ../../shared/gen/cart

replace kuoz/netshop/platform/shared/proto/common => ../../shared/gen/common
