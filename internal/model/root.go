package model

import "net/http"

// RootResponse describes the response of the root endpoint.
type RootResponse struct {

	// The name of the service.
	Service string `json:"service"`

	// The service title.
	Title string `json:"title"`

	// The service version.
	Version string `json:"version"`
}
type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
	Status string      `json:"status"`
}

func SuccessResponse(data interface{}, status int) Response {
	return Response{
		Result: data,
		Status: http.StatusText(status),
	}
}

func ErrorResponse(errStr string, status int) Response {
	return Response{
		Error:  errStr,
		Status: http.StatusText(status),
	}
}
