package grpcapi

import (
	"log"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/grpcapi/actionservice"
	"github.com/thakkarparth007/dalal-street-server/grpcapi/streamservice"
	"github.com/thakkarparth007/dalal-street-server/matchingengine"
	"github.com/thakkarparth007/dalal-street-server/proto_build"
	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var config *utils.Config

var logger *logrus.Entry

func authFunc(ctx context.Context) (context.Context, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "authFunc",
	})
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "Missing context metadata")
	}
	if len(md["bot_secret"]) == 1 {
		if md["bot_secret"][0] == config.BotSecret && len(md["bot_user_id"]) == 1 {
			sess, err := session.Fake()
			if err != nil {
				l.Errorf("Unable to create session for bot")
				return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
			}
			err = sess.Set("UserId", md["bot_user_id"][0])
			ctx = context.WithValue(ctx, "session", sess)
			return ctx, nil
		} else {
			return nil, grpc.Errorf(codes.Unauthenticated, "bot Secret not set")
		}
	}
	if len(md["sessionid"]) != 1 {
		return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
	}
	sess, err := session.Load(md["sessionid"][0])
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
		newSess, err := session.New()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Internal error occurred")
		}
		ctx = context.WithValue(ctx, "session", newSess)
		return handler(ctx, req)
	}

	newCtx, err := authFunc(ctx)
	if err != nil {
		return nil, err
	}

	return handler(newCtx, req)
}

// Init configures the grpcapi package
func Init(conf *utils.Config) {
	config = conf
}

// StartServices starts the Action and Stream services
// It passes on the Matching Engine to Action service.
func StartServices(matchingEngine matchingengine.MatchingEngine, dsm datastreams.Manager) {
	creds, err := credentials.NewServerTLSFromFile(config.GrpcCert, config.GrpcKey)
	if err != nil {
		log.Fatalf("Failed while obtaining TLS certificates. Error: %+v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_auth.StreamServerInterceptor(authFunc), // all routes require authentication
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryAuthInterceptor, // all routes except Login require authentication
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	pb.RegisterDalalActionServiceServer(grpcServer, actionservice.NewDalalActionService(matchingEngine))
	pb.RegisterDalalStreamServiceServer(grpcServer, streamservice.NewDalalStreamService(dsm))

	lis, err := net.Listen("tcp", config.GrpcAddress)
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
