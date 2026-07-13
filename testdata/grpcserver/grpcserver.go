package grpcserver

import (
	"context"
)

type UnimplementedPayoutServer struct{}

type PayoutServer struct{} // want "gRPC controller PayoutServer must embed"

func (s *PayoutServer) GetPayout(ctx context.Context, req *PayoutRequest) (*PayoutResponse, error) {
	return nil, nil
}

type GoodServer struct {
	UnimplementedPayoutServer
}

func (s *GoodServer) GetPayout(ctx context.Context, req *PayoutRequest) (*PayoutResponse, error) {
	return nil, nil
}

type PayoutRequest struct{}
type PayoutResponse struct{}
