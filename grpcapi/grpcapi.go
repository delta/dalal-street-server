package grpcapi

import (
	"log"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/thakkarparth007/dalal-street-server/grpcapi/actionservice"
	"github.com/thakkarparth007/dalal-street-server/grpcapi/streamservice"
	"github.com/thakkarparth007/dalal-street-server/proto_build"
	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/session"
)

func authFunc(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "Missing context metadata")
	}
	if len(md["sessionId"]) != 1 {
		return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
	}
	sess, err := session.Load(md["sessionId"][0])
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
	}
	ctx = context.WithValue(ctx, "session", sess)
	return ctx, nil
}

// allows "Login" requests to pass through unauthenticated. Others require authentication
func unaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	switch req.(type) {
	case *actions_pb.LoginRequest:
		return handler(ctx, req)
	}

	newCtx, err := authFunc(ctx)
	if err != nil {
		return nil, err
	}
	return handler(newCtx, req)
}

func StartServices() {
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_auth.StreamServerInterceptor(authFunc), // all routes require authentication
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryAuthInterceptor, // all routes except require authentication
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	pb.RegisterDalalActionServiceServer(grpcServer, actionservice.NewDalalActionService())
	pb.RegisterDalalStreamServiceServer(grpcServer, streamservice.NewDalalStreamService())

	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("Failed while listening on port 8000. Error: %+v", err)
	}

	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			log.Fatalf("Failed while starting server. Error: %+v", err)
		}
	}()
}
