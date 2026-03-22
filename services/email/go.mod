module netshop/services/email

go 1.24.0

require (
	google.golang.org/grpc v1.76.0
	kuoz/netshop/platform/shared/proto/email v0.0.0
)

require (
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	kuoz/netshop/platform/shared/proto/common v0.0.0 // indirect
)

replace kuoz/netshop/platform/shared/proto/email => ../../shared/gen/email

replace kuoz/netshop/platform/shared/proto/common => ../../shared/gen/common
