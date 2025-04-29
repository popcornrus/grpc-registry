module my-extension-server

go 1.21

require (
	github.com/sirupsen/logrus v1.9.3
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/popcornrus/grpc-registry v0.0.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
)

// Replace with your actual grpc-registry path for local development
replace github.com/popcornrus/grpc-registry => ../grpc-registry
