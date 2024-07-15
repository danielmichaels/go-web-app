package cmd

import (
	"fmt"
	"os"
)

type WorkersCmd struct{}

func (w *WorkersCmd) Run() error {
	_,_ = fmt.Fprintf(os.Stderr, "workers called")
	return nil
}
