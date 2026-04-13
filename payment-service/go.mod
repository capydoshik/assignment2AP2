module payment-service

go 1.25.5

require (
	github.com/ayazb/ap2-generated-contracts v0.0.0
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.12.3
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)

replace github.com/ayazb/ap2-generated-contracts => ../generated-contracts
