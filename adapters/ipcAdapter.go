package adapters

import (
	"context"
	"fmt"
	"github.com/CryoCodec/jim/config"
	"github.com/CryoCodec/jim/core/domain"
	"github.com/CryoCodec/jim/core/ports"
	pb "github.com/CryoCodec/jim/internal/proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"time"
)

type ipcAdapterImpl struct {
	grpcContext *GrpcContext
}

// InstantiateAdapter instantiates an implementation of the IpcPort
func InstantiateAdapter(grpcContext *GrpcContext) ports.IpcPort {
	return &ipcAdapterImpl{grpcContext: grpcContext}
}

// LoadConfigFile causes the server to load the config file.
func (adapter *ipcAdapterImpl) LoadConfigFile(path string) error {
	client := adapter.grpcContext.client
	ctx, cancel := adapter.grpcContext.newCtxWithDefaultTimeout()
	defer cancel()
	reply, err := client.LoadConfigFile(ctx, &pb.LoadRequest{Destination: path})
	if err != nil {
		log.Debugf("Received unexpected error %s", err)
		return err
	}
	if reply.ResponseType == pb.ResponseType_FAILURE {
		log.Debugf("Received domain failure response %s", err)
		return errors.New(reply.Reason)
	}

	return nil
}

// GetMatchingServer asks the server for a matching entry for the query string.
// The server has to be in ready state.
func (adapter *ipcAdapterImpl) GetMatchingServer(query string) (*domain.Match, error) {
	client := adapter.grpcContext.client
	ctx, cancel := adapter.grpcContext.newCtxWithDefaultTimeout()
	defer cancel()

	response, err := client.Match(ctx, &pb.MatchRequest{Query: query})
	if err != nil {
		return nil, err
	}

	server := domain.Server{
		Host:     response.Server.Info.Host,
		Dir:      response.Server.Info.Directory,
		Port:     int(response.Server.Port),
		Username: response.Server.Username,
		Password: response.Server.Password,
	}

	return &domain.Match{Tag: response.Tag,
		Server: server}, nil
}

// GetEntries asks the server for all entries in the config file and returns these.
// The server has to be in ready state.
func (adapter *ipcAdapterImpl) GetEntries(filter *domain.Filter, limit int) (*domain.GroupList, error) {
	client := adapter.grpcContext.client
	ctx, cancel := adapter.grpcContext.newCtxWithDefaultTimeout()
	defer cancel()

	request := createRequestFromFilters(filter, limit)

	response, err := client.List(ctx, request)

	if err != nil {
		return nil, err
	}

	var result domain.GroupList
	for _, pbGroup := range response.Groups {
		var entryList domain.ConnectionList
		for _, entry := range pbGroup.Entries {
			conn := domain.ConnectionInfo{
				Tag:      entry.Tag,
				HostInfo: fmt.Sprintf("%s:%s", entry.Info.Host, entry.Info.Directory),
			}
			entryList = append(entryList, conn)
		}

		domainGroup := domain.Group{
			Title:   pbGroup.Title,
			Entries: entryList,
		}
		result = append(result, domainGroup)
	}

	return &result, nil
}

func createRequestFromFilters(filter *domain.Filter, limit int) *pb.ListRequest {
	pf := &pb.Filter{}
	request := &pb.ListRequest{Filter: pf, Limit: int32(limit)}
	if filter.IsAnyFilterSet() {
		if filter.HasGroupFilter() {
			pf.Group = filter.GroupFilter
		}
		if filter.HasEnvFilter() {
			pf.Env = filter.EnvFilter
		}
		if filter.HasTagFilter() {
			pf.Tag = filter.TagFilter
		}
		if filter.HasHostFilter() {
			pf.Host = filter.HostFilter
		}
		if filter.HasFreeFilter() {
			pf.Free = filter.FreeFilter
		}
	}

	return request
}

// MatchClosestN gets a list of potentially matching entries in the config file
func (adapter *ipcAdapterImpl) MatchClosestN(query string) []string {
	client := adapter.grpcContext.client
	ctx, cancel := adapter.grpcContext.newCtxWithDefaultTimeout()
	defer cancel()
	response, err := client.MatchN(ctx, &pb.MatchNRequest{
		Query:           query,
		NumberOfResults: 3,
	})

	if err != nil {
		return []string{}
	}

	return response.Tags
}

// IsServerReady checks whether the server is ready to serve
func (adapter *ipcAdapterImpl) IsServerReady() bool {
	state, err := adapter.ServerStatus()
	if err != nil {
		return false
	}

	return state.IsReady()
}

// ServerStatus queries and returns the server state.
func (adapter *ipcAdapterImpl) ServerStatus() (*domain.ServerState, error) {
	client := adapter.grpcContext.client
	ctx, cancel := adapter.grpcContext.newCtxWithDefaultTimeout()
	defer cancel()
	response, err := client.GetState(ctx, &pb.StateRequest{})

	if err != nil {
		return nil, err
	}

	switch response.State {
	case pb.StateReply_CONFIG_FILE_REQUIRED:
		return domain.NewServerState(domain.RequiresConfigFile)
	case pb.StateReply_DECRYPTION_REQUIRED:
		return domain.NewServerState(domain.RequiresDecryption)
	case pb.StateReply_READY:
		return domain.NewServerState(domain.Ready)
	default:
		return nil, errors.Errorf("Received unhandled protobuf state: %s", response.State)
	}

}

// AttemptDecryption asks the server to try decryption of the config file with the given password.
func (adapter *ipcAdapterImpl) AttemptDecryption(password []byte) error {
	client := adapter.grpcContext.client
	ctx, cancel := adapter.grpcContext.newTimedCtx(5 * time.Second)
	defer cancel()
	response, err := client.Decrypt(ctx, &pb.DecryptRequest{Password: password})

	if err != nil {
		return err
	}

	if response.ResponseType == pb.ResponseType_FAILURE {
		return errors.New(response.Reason)
	}

	return nil
}

// Close closes the underlying ipc connection
func (adapter *ipcAdapterImpl) Close() error {
	return adapter.grpcContext.Close()
}

type GrpcContext struct {
	client  pb.JimClient
	timeout time.Duration
	conn    *grpc.ClientConn
}

func (ctx *GrpcContext) newCtxWithDefaultTimeout() (context.Context, context.CancelFunc) {
	rootCtx := context.Background()
	return context.WithTimeout(rootCtx, ctx.timeout)
}

func (ctx *GrpcContext) newTimedCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	rootCtx := context.Background()
	return context.WithTimeout(rootCtx, timeout)
}

func (ctx *GrpcContext) Close() error {
	err := ctx.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

// InitializeGrpcContext creates an ipc client, which may be used to
// write and receive data from a unix domain socket/named pipe
func InitializeGrpcContext() *GrpcContext {
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		log.Debugf("Dial called with addr:%s and protocol:%s", addr, config.Protocol)
		return net.Dial(config.Protocol, addr)
	}

	conn, err := grpc.Dial(config.GetSocketAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer), withClientUnaryInterceptor())
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewJimClient(conn)

	return &GrpcContext{client: client, timeout: 3 * time.Second, conn: conn}
}

func withClientUnaryInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(loggingInterceptor)
}

func loggingInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Logic before invoking the invoker

	start := time.Now()
	// Calls the invoker to execute RPC
	err := invoker(ctx, method, req, reply, cc, opts...)
	// Logic after invoking the invoker
	log.Debugf("Invoked RPC method=%s; Duration=%s; Error=%v", method, time.Since(start), err)
	return err
}
