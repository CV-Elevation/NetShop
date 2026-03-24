module netshop/services/aiassistant

go 1.26.1

replace kuoz/netshop/platform/shared/proto/aiassistant => ../../shared/gen/aiassistant

replace kuoz/netshop/platform/shared/proto/common => ../../shared/gen/common

replace kuoz/netshop/platform/shared/proto/product => ../../shared/gen/product

require (
	google.golang.org/grpc v1.79.3
	kuoz/netshop/platform/shared/proto/aiassistant v0.0.0-00010101000000-000000000000
	kuoz/netshop/platform/shared/proto/common v0.0.0-00010101000000-000000000000
	kuoz/netshop/platform/shared/proto/product v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
