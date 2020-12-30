package list

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/google/subcommands"

	"github.com/loov/goda/pkggraph"
	"github.com/loov/goda/pkgset"
	"github.com/loov/goda/templates"
)

type Command struct {
	printStandard bool

	noAlign bool
	header  string
	format  string
}

func (*Command) Name() string     { return "list" }
func (*Command) Synopsis() string { return "List packages" }
func (*Command) Usage() string {
	return `list <expr>:
	List packages using an expression.

	See "help expr" for further information about expressions.
`
}

func (cmd *Command) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&cmd.printStandard, "std", false, "print std packages")

	f.BoolVar(&cmd.noAlign, "noalign", false, "disable aligning tabs")
	f.StringVar(&cmd.header, "h", "", "header for the table\nautomatically derives from format, when empty, use \"-\" to skip")
	f.StringVar(&cmd.format, "f", "{{.ID}}", "formatting")
}

func (cmd *Command) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing package names\n")
		return subcommands.ExitUsageError
	}

	t, err := templates.Parse(cmd.format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid format string: %v\n", err)
		return subcommands.ExitFailure
	}

	result, err := pkgset.Calc(ctx, f.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return subcommands.ExitFailure
	}
	if !cmd.printStandard {
		result = pkgset.Subtract(result, pkgset.Std())
	}

	graph := pkggraph.From(result)

	var w io.Writer = os.Stdout
	if !cmd.noAlign {
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	}
	if cmd.header != "-" {
		if cmd.header == "" {
			rx := regexp.MustCompile(`(\{\{\s*\.?|\s*\}\})`)
			cmd.header = rx.ReplaceAllString(cmd.format, "")
		}
		fmt.Fprintln(w, cmd.header)
	}
	for _, p := range graph.Sorted {
		err := t.Execute(w, p)
		fmt.Fprintln(w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
		}
	}
	if w, ok := w.(interface{ Flush() error }); ok {
		w.Flush()
	}

	return subcommands.ExitSuccess
}
