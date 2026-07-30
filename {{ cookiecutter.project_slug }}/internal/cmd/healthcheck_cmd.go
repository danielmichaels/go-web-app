package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// HealthcheckCmd probes this process over loopback. It exists so a distroless
// image — which has neither a shell nor curl — can still declare a Docker
// HEALTHCHECK.
type HealthcheckCmd struct {
	Timeout time.Duration `help:"How long to wait for a response." default:"3s"`
}

// Run reads SERVER_PORT straight from the environment rather than loading the
// full configuration: a health probe must not fail because some unrelated
// setting is missing.
func (h *HealthcheckCmd) Run() error {
	port := 9898
	if v := os.Getenv("SERVER_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("healthcheck: invalid SERVER_PORT %q: %w", v, err)
		}
		port = p
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: %s returned %s", url, resp.Status)
	}
	return nil
}
