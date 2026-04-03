module netshop/services/aiassistant

go 1.26.1

replace kuoz/netshop/platform/shared/proto/aiassistant => ../../shared/gen/aiassistant

replace kuoz/netshop/platform/shared/proto/common => ../../shared/gen/common

replace kuoz/netshop/platform/shared/proto/product => ../../shared/gen/product

require (
	github.com/jackc/pgx/v5 v5.7.6
	github.com/sashabaranov/go-openai v1.41.2
	google.golang.org/grpc v1.79.3
	kuoz/netshop/platform/shared/proto/aiassistant v0.0.0-00010101000000-000000000000
	kuoz/netshop/platform/shared/proto/common v0.0.0-00010101000000-000000000000
	kuoz/netshop/platform/shared/proto/product v0.0.0-00010101000000-000000000000
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
