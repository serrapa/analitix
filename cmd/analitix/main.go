package main

import (
	"context"
	"fmt"
	"os"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/formatter"
	"github.com/projectdiscovery/gologger/levels"
	"github.com/serrapa/analitix/internal/runner"
	"github.com/serrapa/analitix/pkg/output"
)

func main() {
	options, err := runner.ParseOptions(os.Args[1:])
	if err != nil {
		gologger.Fatal().Msgf("%s", err)
	}

	configureLogger(options)

	if options.Version {
		fmt.Printf("analitix %s\n", runner.Version)
		return
	}

	if options.Update {
		gologger.Info().Msg("update flow is not implemented yet")
		return
	}

	if !options.Silent && !options.JSON && !options.JSONL && !options.ListSources {
		fmt.Print(runner.Banner())
	}

	writer, closeWriter, err := output.Open(options.Output, os.Stdout)
	if err != nil {
		gologger.Fatal().Msgf("%s", err)
	}
	defer closeWriter()

	r, err := runner.New(options)
	if err != nil {
		gologger.Fatal().Msgf("%s", err)
	}

	if err := r.Run(context.Background(), writer); err != nil {
		gologger.Fatal().Msgf("%s", err)
	}
}

func configureLogger(options *runner.Options) {
	gologger.DefaultLogger.SetFormatter(formatter.NewCLI(options.NoColor))
	if !options.JSON && !options.JSONL {
		gologger.DefaultLogger.SetWriter(stdoutLogWriter{})
	}
	switch {
	case options.Silent:
		gologger.DefaultLogger.SetMaxLevel(levels.LevelSilent)
	case (options.JSON || options.JSONL) && !options.Verbose:
		gologger.DefaultLogger.SetMaxLevel(levels.LevelWarning)
	case options.Verbose:
		gologger.DefaultLogger.SetMaxLevel(levels.LevelVerbose)
	default:
		gologger.DefaultLogger.SetMaxLevel(levels.LevelInfo)
	}
}

type stdoutLogWriter struct{}

func (stdoutLogWriter) Write(data []byte, _ levels.Level) {
	_, _ = os.Stdout.Write(data)
	_, _ = os.Stdout.WriteString("\n")
}
