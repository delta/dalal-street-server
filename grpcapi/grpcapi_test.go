package grpcapi

import (
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/thakkarparth007/dalal-street-server/proto_build"
	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"

	"google.golang.org/grpc"
)

func init() {
	StartServices("../tls_keys/test/server.crt", "../tls_keys/test/server.key", nil)
}

func getConnection(t *testing.T) *grpc.ClientConn {
	creds, err := credentials.NewClientTLSFromFile("../tls_keys/test/server.crt", "")
	if err != nil {
		t.Fatalf("Failed getting server's public key. Error %+v", err)
	}

	conn, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(creds))
	if err != nil {
		t.Fatalf("Failed connecting to gRPC Server on port 8000. Error: %+v", err)
	}

	return conn
}

func Test_Authentication(t *testing.T) {
	conn := getConnection(t)

	actionClient := pb.NewDalalActionServiceClient(conn)
	streamClient := pb.NewDalalStreamServiceClient(conn)

	var statusCode *status.Status
	var err error

	// Login shouldn't fail with Unauthenticated error.
	// It should fail with InvalidCredentialsError
	loginReq := &actions_pb.LoginRequest{
		Email:    "test@test.com",
		Password: "test",
	}
	loginRes, err := actionClient.Login(context.Background(), loginReq)
	statusCode, _ = status.FromError(err)
	if statusCode.Code() == codes.Unauthenticated {
		t.Fatalf("Unexpected: Login request gave Unauthenticated error %+v", err)
	}
	if loginRes.GetStatusCode() != actions_pb.LoginResponse_InvalidCredentialsError {
		t.Fatalf("Unexpected: Login request failed with %+v", loginRes)
	}

	buyStocksFromExchangeReq := &actions_pb.BuyStocksFromExchangeRequest{1, 1}
	_, err = actionClient.BuyStocksFromExchange(context.Background(), buyStocksFromExchangeReq)
	statusCode, _ = status.FromError(err)
	if statusCode.Code() != codes.Unauthenticated {
		t.Fatalf("Expected Unauthenticated error for non-login request. Got %+v", err)
	}

	subId := &datastreams_pb.SubscriptionId{
		"1",
		datastreams_pb.DataStreamType_MARKET_DEPTH,
	}
	res, err := streamClient.GetMarketDepthUpdates(context.Background(), subId)
	statusCode, _ = status.FromError(err)
	if statusCode.Code() != codes.OK {
		t.Fatalf("Unpexected error while connecting to non-login stream request. Got %+v", err)
	}
	realRes, err := res.Recv()
	statusCode, _ = status.FromError(err)
	if statusCode.Code() != codes.Unauthenticated {
		t.Fatalf("Expected Unauthenticated error for non-login stream request. Got %+v", realRes)
	}
}
