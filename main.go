package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
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

	if args.CPUProfile != nil {
		f, err := os.Create(*args.CPUProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error closing CPU profile: %v\n", err)
			}
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "could not start CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
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

	if args.MemProfile != nil {
		f, err := os.Create(*args.MemProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create memory profile: %v\n", err)
		} else {
			defer func() {
				if err := f.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "error closing memory profile: %v\n", err)
				}
			}()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "could not write memory profile: %v\n", err)
			}
		}
	}

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
