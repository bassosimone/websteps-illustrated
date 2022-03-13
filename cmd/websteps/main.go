// Command websteps is a websteps client.
package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/bassosimone/getoptx"
	"github.com/bassosimone/websteps-illustrated/internal/engine/experiment/websteps"
	"github.com/bassosimone/websteps-illustrated/internal/measurex"
	"github.com/bassosimone/websteps-illustrated/internal/runtimex"
)

type CLI struct {
	Backend string   `doc:"backend URL (default: use OONI backend)" short:"b"`
	Deep    bool     `doc:"causes websteps to scan more IP addresses and follow more redirects"`
	Help    bool     `doc:"prints this help message" short:"h"`
	Input   []string `doc:"add URL to list of URLs to crawl" short:"i"`
	Output  string   `doc:"file where to write output (default: report.jsonl)" short:"o"`
	Verbose bool     `doc:"enable verbose mode" short:"v"`
}

func main() {
	opts := &CLI{
		Backend: "wss://0.th.ooni.org/websteps/v1/th",
		Deep:    false,
		Help:    false,
		Input:   []string{},
		Output:  "report.jsonl",
		Verbose: false,
	}
	parser := getoptx.MustNewParser(opts, getoptx.NoPositionalArguments())
	parser.MustGetopt(os.Args)
	if opts.Help {
		parser.PrintUsage(os.Stdout)
		os.Exit(0)
	}
	if len(opts.Input) < 1 {
		log.Fatal("no input provided (try `./websteps --help' for more help)")
	}
	if opts.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	filep, err := os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Fatal("cannot create output file")
	}
	begin := time.Now()
	ctx := context.Background()
	clientOptions := &measurex.Options{
		MaxAddressesPerFamily: measurex.DefaultMaxAddressPerFamily,
		MaxCrawlerDepth:       measurex.DefaultMaxCrawlerDepth,
	}
	if opts.Deep {
		clientOptions.MaxAddressesPerFamily = 32
		clientOptions.MaxCrawlerDepth = 11
	}
	clnt := websteps.StartClient(ctx, log.Log, nil, nil, opts.Backend, clientOptions)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go submitInput(ctx, wg, clnt, opts)
	processOutput(begin, filep, clnt)
	wg.Wait()
	if err := filep.Close(); err != nil {
		log.WithError(err).Fatal("cannot close output file")
	}
}

func submitInput(ctx context.Context, wg *sync.WaitGroup, clnt *websteps.Client, opts *CLI) {
	defer close(clnt.Input)
	defer wg.Done()
	for _, input := range opts.Input {
		clnt.Input <- input
		if ctx.Err() != nil {
			return
		}
	}
}

// result is the result of running websteps on an input URL.
type result struct {
	// TestKeys contains the experiment test keys.
	TestKeys *websteps.ArchivalTestKeys `json:"test_keys"`
}

func processOutput(begin time.Time, filep io.Writer, clnt *websteps.Client) {
	for tkoe := range clnt.Output {
		if err := tkoe.Err; err != nil {
			log.Warn(err.Error())
			continue
		}
		r := &result{TestKeys: tkoe.TestKeys.ToArchival(begin)}
		data, err := json.Marshal(r)
		runtimex.PanicOnError(err, "json.Marshal failed")
		data = append(data, '\n')
		if _, err := filep.Write(data); err != nil {
			log.WithError(err).Fatal("cannot write output file")
		}
	}
}
