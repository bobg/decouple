package fab

import (
	"context"
	"os"

	"github.com/bobg/fab"
	"github.com/bobg/fab/deps"
	"github.com/pkg/errors"
)

var (
	Cover = fab.Deps(showCoverage, computeCoverage)
	Clean = fab.Clean("cover.out")

	showCoverage = fab.Command("go tool cover -html cover.out")
)

var computeCoverage = fab.Register("computeCoverage", "compute coverage profile", fab.F(func(ctx context.Context) error {
	in, err := deps.Go(".", true)
	if err != nil {
		return errors.Wrap(err, "in deps.Go")
	}
	filesTarget := fab.Files{
		Target: fab.Command("go test -coverprofile cover.out ./...", fab.CmdStdout(os.Stdout)),
		In:     in,
		Out:    []string{"cover.out"},
	}
	return filesTarget.Run(ctx)
}))
