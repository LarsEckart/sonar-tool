package cmd

import (
	"context"
	"io"

	cli "github.com/urfave/cli/v3"
)

func New(version string, stdout, stderr io.Writer) *cli.Command {
	cli.VersionFlag = &cli.BoolFlag{
		Name:        "version",
		Usage:       "print the version",
		HideDefault: true,
		Local:       true,
	}

	return &cli.Command{
		Name:        "sonar-issues",
		Usage:       "Query SonarCloud projects and issues",
		UsageText:   "sonar-issues [global flags] <command> [subcommand] [flags]",
		Description: "Small, script-friendly SonarCloud CLI with secure local auth storage.",
		Version:     version,
		Writer:      stdout,
		ErrWriter:   stderr,
		Suggest:     true,
		Flags:       globalFlags(),
		Commands: []*cli.Command{
			newAuthCommand(),
			newProjectsCommand(),
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			_, err := outputMode(cmd)
			return ctx, err
		},
	}
}

func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "host",
			Aliases: []string{"s"},
			Usage:   "Base URL for SonarCloud or a compatible SonarQube host",
			Value:   "https://sonarcloud.io",
			Sources: cli.EnvVars("SONAR_HOST_URL"),
		},
		&cli.StringFlag{
			Name:    "org",
			Aliases: []string{"o"},
			Usage:   "Default organization key",
			Sources: cli.EnvVars("SONAR_ORG"),
		},
		&cli.IntFlag{
			Name:    "timeout",
			Usage:   "HTTP timeout in seconds",
			Value:   30,
			Sources: cli.EnvVars("SONAR_TIMEOUT"),
		},
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Print structured JSON output",
		},
		&cli.BoolFlag{
			Name:    "markdown",
			Aliases: []string{"md"},
			Usage:   "Print markdown output",
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable color in human output",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Suppress non-data success output",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Usage:   "Print more diagnostics to stderr",
		},
	}
}
