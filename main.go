package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/nixys/nxs-data-anonymizer/ctx"
	"github.com/nixys/nxs-data-anonymizer/misc"
	"github.com/nixys/nxs-data-anonymizer/routines/anonymizer"
	"github.com/nixys/nxs-data-anonymizer/routines/generator"

	_ "github.com/go-sql-driver/mysql"
	appctx "github.com/nixys/nxs-go-appctx/v3"
)

func main() {

	// Check for generate mode
	args, err := ctx.ArgsRead()
	if err != nil {
		switch err {
		case misc.ErrArgSuccessExit:
			os.Exit(0)
		default:
			os.Exit(1)
		}
	}

	if args.Generate != nil {
		if err := generator.Run(args.ConfigPath, *args.Generate); err != nil {
			fmt.Fprintf(os.Stderr, "generator error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	err = appctx.Init(context.Background()).
		RoutinesSet(
			map[string]appctx.RoutineParam{
				"anonymizer": {
					Handler: anonymizer.Runtime,
				},
			},
		).
		ValueInitHandlerSet(ctx.AppCtxInit).
		SignalsSet([]appctx.SignalsParam{
			{
				Signals: []os.Signal{
					syscall.SIGTERM,
				},
				Handler: sigHandlerTerm,
			},
		}).
		Run()
	if err != nil {
		switch err {
		case misc.ErrArgSuccessExit:
			os.Exit(0)
		default:
			os.Exit(1)
		}
	}
}

func sigHandlerTerm(sig appctx.Signal) {
	sig.Shutdown(nil)
}
