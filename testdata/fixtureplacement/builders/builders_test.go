package builders

import "testdata/fixtureplacement"

func createRequest() fixtureplacement.CreateRequest {
	return fixtureplacement.CreateRequest{Name: "shared"}
}
