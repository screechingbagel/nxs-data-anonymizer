package ctx

import (
	"fmt"
	"os"

	"github.com/nixys/nxs-data-anonymizer/misc"
	"github.com/pborman/getopt/v2"
)

const (
	confPathDefault = "/nxs-data-anonymizer.conf"
)

var version string

// Args contains arguments value read from command line
type Args struct {
	ConfigPath string
	LogFormat  LogFormat
	Input      *string
	Output     *string
	Cleanup    bool
	DBType     DBType
	Generate   *string
}

// ArgsRead reads arguments from command line
func ArgsRead() (Args, error) {

	args := getopt.New()

	helpFlag := args.BoolLong(
		"help",
		'h',
		"Show help")

	versionFlag := args.BoolLong(
		"version",
		'v',
		"Show program version")

	confPath := args.StringLong(
		"conf",
		'c',
		confPathDefault,
		"Config file path")

	input := args.StringLong(
		"input",
		'i',
		"",
		"Input file. If not set `stdin` is used")

	logformat := args.EnumLong(
		"log-format",
		'l',
		[]string{
			string(LogFormatJSON),
			string(LogFormatPlain),
		},
		string(LogFormatJSON),
		fmt.Sprintf("Log file format. Values `%s` or `%s` are available", LogFormatJSON, LogFormatPlain),
	)

	output := args.StringLong(
		"output",
		'o',
		"",
		"Output file. If not set `stdout` is used")

	generate := args.StringLong(
		"generate",
		'g',
		"",
		"Generate optimized Go code to specified file")

	dbType := args.EnumLong(
		"type",
		't',
		[]string{
			string(DBTypeMySQL),
			string(DBTypePgSQL),
		},
		"",
		fmt.Sprintf("Database type you need to operate. Values `%s` or `%s` are available", DBTypePgSQL, DBTypeMySQL),
	)

	cleanup := args.BoolLong(
		"cleanup",
		'C',
		"Clean up destination database (experimental). Available only for MySQL")

	args.Parse(os.Args)

	/* Show help */
	if *helpFlag {
		argsHelp(args)
		return Args{}, misc.ErrArgSuccessExit
	}

	/* Show version */
	if *versionFlag {
		argsVersion()
		return Args{}, misc.ErrArgSuccessExit
	}

	// If generating code, we don't need DBType validation strictness yet,
	// but config path is required.
	if !args.IsSet("type") && !args.IsSet("generate") {
		fmt.Println("args: 'type' option must be specified")
		return Args{}, misc.ErrConig
	}

	return Args{
		ConfigPath: *confPath,
		LogFormat:  LogFormat(*logformat),
		Input: func() *string {
			if args.IsSet("input") {
				return input
			}
			return nil
		}(),
		Output: func() *string {
			if args.IsSet("output") {
				return output
			}
			return nil
		}(),
		Generate: func() *string {
			if args.IsSet("generate") {
				return generate
			}
			return nil
		}(),
		Cleanup: *cleanup,
		DBType:  DBType(*dbType),
	}, nil
}

func argsHelp(args *getopt.Set) {

	additionalDescription := `

Additional description

  Tool for anonymizing PostgreSQL and MySQL databases' dump
`

	args.PrintUsage(os.Stdout)
	fmt.Println(additionalDescription)
}

func argsVersion() {
	fmt.Println(version)
}
