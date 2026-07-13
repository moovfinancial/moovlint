package grpcstatus

import (
	"context"
	"fmt"
	"log"
)

type UnimplementedPayoutServer struct{}

type PayoutServer struct {
	UnimplementedPayoutServer
	logger log.Logger
}

func (s *PayoutServer) BadReturn(ctx context.Context, req *PayoutRequest) (*PayoutResponse, error) {
	return nil, fmt.Errorf("something went wrong") // want "gRPC handler error must be returned through GrpcErrorStatus"
}

func (s *PayoutServer) GoodReturn(ctx context.Context, req *PayoutRequest) (*PayoutResponse, error) {
	err := fmt.Errorf("something went wrong")
	return nil, GrpcErrorStatus(s.logger, err)
}

func (s *PayoutServer) GoodNil(ctx context.Context, req *PayoutRequest) (*PayoutResponse, error) {
	return nil, nil
}

func GrpcErrorStatus(logger log.Logger, err error) error {
	return err
}

type PayoutRequest struct{}
type PayoutResponse struct{}
