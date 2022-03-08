// Command websteps is a websteps client.
package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/bassosimone/getoptx"
	"github.com/bassosimone/websteps-illustrated/internal/engine/experiment/websteps"
	"github.com/bassosimone/websteps-illustrated/internal/runtimex"
)

type CLI struct {
	Backend string   `doc:"backend URL (default: http://127.0.0.1:9876)" short:"b"`
	Help    bool     `doc:"prints this help message" short:"h"`
	Input   []string `doc:"add URL to list of URLs to crawl" short:"i"`
	Output  string   `doc:"file where to write output (default: report.jsonl)" short:"o"`
	Verbose bool     `doc:"enable verbose mode" short:"v"`
}

func main() {
	opts := &CLI{
		Backend: "http://127.0.0.1:9876/",
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
	if opts.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	filep, err := os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Fatal("cannot create output file")
	}
	begin := time.Now()
	ctx := context.Background()
	clnt := websteps.StartClient(ctx, log.Log, http.DefaultClient, opts.Backend, "")
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

// TODO(bassosimone): websteps.TestKeys is not the correct name.

// result is the result of running websteps on an input URL.
type result struct {
	// TestKeys contains the experiment test keys.
	TestKeys *testKeys `json:"test_keys"`
}

type testKeys struct {
	// Steps contains the steps we performed.
	Steps []*websteps.ArchivalTestKeys `json:"steps"`
}

func processOutput(begin time.Time, filep io.Writer, clnt *websteps.Client) {
	r := &result{
		TestKeys: &testKeys{},
	}
	for tkor := range clnt.Output {
		if err := tkor.Err; err != nil {
			log.Warn(err.Error())
			continue
		}
		r.TestKeys.Steps = append(r.TestKeys.Steps, tkor.TestKeys.ToArchival(begin))
	}
	data, err := json.Marshal(r)
	runtimex.PanicOnError(err, "json.Marshal failed")
	data = append(data, '\n')
	if _, err := filep.Write(data); err != nil {
		log.WithError(err).Fatal("cannot write output file")
	}
}
