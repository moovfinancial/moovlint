// Package payoutpb models a generated protobuf package: a *Server service
// interface and its request and response messages.
package payoutpb

import "context"

// PayoutServer is the generated service interface.
type PayoutServer interface {
	GetPayout(context.Context, *GetPayoutRequest) (*GetPayoutResponse, error)
}

// UnimplementedPayoutServer must be embedded for forward compatibility.
type UnimplementedPayoutServer struct{}

func (UnimplementedPayoutServer) GetPayout(context.Context, *GetPayoutRequest) (*GetPayoutResponse, error) {
	return nil, nil
}

type GetPayoutRequest struct{}

type GetPayoutResponse struct{}
