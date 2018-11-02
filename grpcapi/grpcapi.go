package grpcapi

import (
	"log"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/delta/dalal-street-server/datastreams"
	"github.com/delta/dalal-street-server/grpcapi/actionservice"
	"github.com/delta/dalal-street-server/grpcapi/streamservice"
	"github.com/delta/dalal-street-server/matchingengine"
	"github.com/delta/dalal-street-server/proto_build"
	"github.com/delta/dalal-street-server/proto_build/actions"
	"github.com/delta/dalal-street-server/session"
	"github.com/delta/dalal-street-server/utils"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
)

var (
	config *utils.Config
	logger *logrus.Entry

	grpcServer    *grpc.Server
	wrappedServer *grpcweb.WrappedGrpcServer
)

func authFunc(ctx context.Context) (context.Context, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "authFunc",
	})

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "Missing context metadata")
	}
	// handle bot related request specially - create a fake session, since we don't
	// want bot requests to pollute the sessions database. The bots will make stateless
	// requests
	if len(md["bot_secret"]) == 1 {
		if md["bot_secret"][0] == config.BotSecret && len(md["bot_user_id"]) == 1 {
			sess, err := session.Fake()
			if err != nil {
				l.Errorf("Unable to create session for bot")
				return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
			}

			sess.Set("userId", md["bot_user_id"][0])
			ctx = context.WithValue(ctx, "session", sess)
			return ctx, nil
		}

		l.Warnf("Invalid bot request. Got %+v", md)
		return nil, grpc.Errorf(codes.Unauthenticated, "bot secret not set")
	}

	// regular requests
	if len(md["sessionid"]) != 1 {
		return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
	}

	sess, err := session.Load(md["sessionid"][0])
	if err != nil {
		return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
	}
	err = sess.Touch() // ignore the error here.
	if err != nil {
		l.Errorf("Got error while touching the session. Suppressing. %+v", err)
	}

	ctx = context.WithValue(ctx, "session", sess)
	return ctx, nil
}

// allows "Login" requests to pass through unauthenticated. Others require authentication
func unaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "unaryAuthInterceptor",
	})

	switch req.(type) {
	case *actions_pb.LoginRequest:
		// if it is a bots request, don't create a new session, even for login requests
		// bots requests always have a fake session. authFunc handles that.
		md, ok := metadata.FromIncomingContext(ctx)
		if ok && len(md["bot_secret"]) > 0 {
			break
		}

		var sess session.Session
		var err error
		if len(md["sessionid"]) == 0 {
			sess, err = session.New()
			if err != nil {
				// Need to refactor this
				if config.Stage == "Dev" || config.Stage == "Test" || config.Stage == "Docker" {
					return nil, status.Errorf(codes.Internal, "Internal error occurred: %+v", err)
				}
				return nil, status.Errorf(codes.Internal, "Internal error occurred")
			}
		} else {
			sess, err = session.Load(md["sessionid"][0])
			if err != nil {
				return nil, grpc.Errorf(codes.Unauthenticated, "Invalid session id")
			}

			err2 := sess.Touch() // ignore the error here.
			if err2 != nil {
				l.Errorf("Got error while touching the session. Suppressing. %+v", err2)
			}
		}
		ctx = context.WithValue(ctx, "session", sess)

		return handler(ctx, req)
	case *actions_pb.RegisterRequest:
		// Fake because a session starts only once the user logs in.
		newSess, err := session.Fake()
		if err != nil {
			if config.Stage == "Dev" || config.Stage == "Test" || config.Stage == "Docker" {
				return nil, status.Errorf(codes.Internal, "Internal error occurred: %+v", err)
			}
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

func streamAuthInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	// no login for StockPrices
	if strings.Contains(info.FullMethod, "StockPrices") {
		return handler(srv, stream)
	} else if strings.Contains(info.FullMethod, "MarketEvents") {
		return handler(srv, stream)
	}

	newCtx, err := authFunc(stream.Context())
	if err != nil {
		return err
	}

	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx
	return handler(srv, wrapped)
}

// Init configures and initalizes the grpcapi package
func Init(conf *utils.Config, matchingEngine matchingengine.MatchingEngine, dsm datastreams.Manager) {
	config = conf
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "grpcapi",
	})

	creds, err := credentials.NewServerTLSFromFile(config.TLSCert, config.TLSKey)
	if err != nil {
		log.Fatalf("Failed while obtaining TLS certificates. Error: %+v", err)
	}

	grpcServer = grpc.NewServer(
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamAuthInterceptor, // all streams expect StockPrices, MarketEvents require authentication
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryAuthInterceptor, // all routes except Login require authentication
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	pb.RegisterDalalActionServiceServer(grpcServer, actionservice.NewDalalActionService(matchingEngine))
	pb.RegisterDalalStreamServiceServer(grpcServer, streamservice.NewDalalStreamService(dsm))

	wrappedServer = grpcweb.WrapServer(grpcServer)
}

// Handler func to handle incoming grpc requests
// Checks the request type and calls the appropriate handler
func GrpcHandlerFunc(resp http.ResponseWriter, req *http.Request) {
	if wrappedServer.IsGrpcWebRequest(req) {
		log.Printf("Got grpc web request")
		wrappedServer.ServeHTTP(resp, req)
	} else {
		grpcServer.ServeHTTP(resp, req)
	}
}
