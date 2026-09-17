package grpcserver

import (
	"context"

	"testdata/payoutpb"
)

// BadController implements payoutpb.PayoutServer without embedding the
// generated Unimplemented type.
type BadController struct{} // want "gRPC controller BadController must embed"

func (s *BadController) GetPayout(ctx context.Context, req *payoutpb.GetPayoutRequest) (*payoutpb.GetPayoutResponse, error) {
	return nil, nil
}

// GoodController embeds the generated Unimplemented type.
type GoodController struct {
	payoutpb.UnimplementedPayoutServer
}

func (s *GoodController) GetPayout(ctx context.Context, req *payoutpb.GetPayoutRequest) (*payoutpb.GetPayoutResponse, error) {
	return nil, nil
}

// PartialController implements one server method and still needs the embed.
type PartialController struct{} // want "gRPC controller PartialController must embed"

func (s *PartialController) GetPayout(ctx context.Context, req *payoutpb.GetPayoutRequest) (*payoutpb.GetPayoutResponse, error) {
	return nil, nil
}

// PlainService has a handler-shaped method with a same-package request type.
// It is not a gRPC controller.
type PlainService struct {
	logger string
}

type localRequest struct{}
type localResponse struct{}

func (s *PlainService) Fetch(ctx context.Context, req *localRequest) (*localResponse, error) {
	return nil, nil
}

// PlainRepo has a handler-shaped method whose request type comes from another
// package, but that package defines no *Server interface with that method.
// It is not a gRPC controller.
type PlainRepo struct{}

func (r *PlainRepo) GetPayout(ctx context.Context, id string) (*payoutpb.GetPayoutResponse, error) {
	return nil, nil
}

// storage accepts a generated request but returns a storage model. It is not
// a gRPC controller because its method does not match PayoutServer exactly.
type storage struct{}

type storedPayout struct{}

func (s *storage) GetPayout(ctx context.Context, req *payoutpb.GetPayoutRequest) (*storedPayout, error) {
	return nil, nil
}

// UnrelatedMethod takes a pb request but does not match any server method
// name. It is not a gRPC controller.
type UnrelatedMethod struct{}

func (u *UnrelatedMethod) Audit(ctx context.Context, req *payoutpb.GetPayoutRequest) (*payoutpb.GetPayoutResponse, error) {
	return nil, nil
}
