package v1

import "context"

type UnimplementedWidgetsServer struct{}

type CreateRequest struct {
	Name string
}

type CreateResponse struct{}

type Widgets_UploadServer interface {
	Context() context.Context
	SendAndClose(*CreateResponse) error
}
