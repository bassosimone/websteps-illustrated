// Command thd is the test helper daemon.
package main

import (
	"context"
	"net/http"
	"os"

	"github.com/apex/log"
	"github.com/bassosimone/getoptx"
	"github.com/bassosimone/websteps-illustrated/internal/engine/experiment/websteps"
	"github.com/bassosimone/websteps-illustrated/internal/measurex"
	"github.com/bassosimone/websteps-illustrated/internal/model"
)

type CLI struct {
	Address     string `doc:"address where to listen (default: \":9876\")" short:"A"`
	CacheDir    string `doc:"directory where to store cache (default: empty)" short:"C"`
	Help        bool   `doc:"prints this help message" short:"h"`
	MostlyCache bool   `doc:"never expire cache entries and keep adding to the cache"`
	User        string `doc:"user to drop privileges to (Linux only; default: nobody)" short:"u"`
	Verbose     bool   `doc:"enable verbose mode" short:"v"`
}

// getopt parses command line options.
func getopt() *CLI {
	opts := &CLI{
		Address:     ":9876",
		CacheDir:    "",
		Help:        false,
		MostlyCache: false,
		User:        "nobody",
		Verbose:     false,
	}
	parser := getoptx.MustNewParser(opts, getoptx.NoPositionalArguments())
	parser.MustGetopt(os.Args)
	if opts.Help {
		parser.PrintUsage(os.Stdout)
		os.Exit(0)
	}
	if opts.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	return opts
}

// maybeOpenCache opens the cache if we configured a cache. Otherwise
// this function returns a nil pointer indicating there's no cache.
func maybeOpenCache(ctx context.Context, opts *CLI) (*measurex.Cache, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	if opts.CacheDir == "" {
		return nil, cancel
	}
	cache := measurex.NewCache(opts.CacheDir)
	olog := measurex.NewOperationLogger(log.Log, "trimming the cache")
	cache.Trim()
	olog.Stop(nil)
	cache.StartTrimmer(ctx)
	return cache, cancel
}

func main() {
	opts := getopt()
	dropprivileges(log.Log, opts.User) // must drop before touching the disk
	cache, cancel := maybeOpenCache(context.Background(), opts)
	defer cancel()
	thOptions := &websteps.THHandlerOptions{
		Logger: log.Log,
		MeasurerFactory: func(logger model.Logger,
			options *measurex.Options) (measurex.AbstractMeasurer, error) {
			lib := measurex.NewDefaultLibrary(logger)
			mx := measurex.NewMeasurerWithOptions(logger, lib, options)
			if cache == nil {
				return mx, nil
			}
			var cpp measurex.CachingPolicy
			switch opts.MostlyCache {
			case true:
				cpp = measurex.CachingForeverPolicy()
			case false:
				cpp = measurex.ReasonableCachingPolicy()
			}
			cmx := measurex.NewCachingMeasurer(mx, logger, cache, cpp)
			return cmx, nil
		},
		Resolvers: nil,
		Saver:     nil,
	}
	thh := websteps.NewTHHandler(thOptions)
	http.Handle("/", thh)
	log.Infof("Listening at: \"%s\"", opts.Address)
	http.ListenAndServe(opts.Address, nil)
}
