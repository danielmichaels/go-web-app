{% if cookiecutter.use_river %}
package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

type ExampleArgs struct {
	Message string `json:"message"`
}

func (ExampleArgs) Kind() string { return "example" }

type ExampleWorker struct {
	river.WorkerDefaults[ExampleArgs]
}

func (w *ExampleWorker) Work(ctx context.Context, job *river.Job[ExampleArgs]) error {
	slog.InfoContext(ctx, "example job", "message", job.Args.Message)
	return nil
}
{% endif %}
