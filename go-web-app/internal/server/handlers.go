package server

import (
	"context"
	"github.com/danielmichaels/go-web-app/internal/version"
)

func (app *Server) handleHealthzGet(_ context.Context, _ *struct{}) (*struct{}, error) {
	return nil, nil
}

type VersionOutput struct {
	Body struct {
		Version string `json:"version" example:"1.0.0" doc:"Version of the API"`
	}
}

func (app *Server) handleVersionGet(_ context.Context, _ *struct{}) (*VersionOutput, error) {
	v := version.Get()
	resp := &VersionOutput{}
	resp.Body.Version = v
	return resp, nil
}
