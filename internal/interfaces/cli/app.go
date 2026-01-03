package cli

import (
	"github.com/felixgeelhaar/bridge/internal/interfaces/cli/commands"
	"github.com/urfave/cli/v2"
)

// NewApp creates the CLI application.
func NewApp() *cli.App {
	app := &cli.App{
		Name:    "bridge",
		Usage:   "AI workflow orchestration and governance platform",
		Version: "0.1.0",
		Commands: []*cli.Command{
			commands.InitCommand(),
			commands.ValidateCommand(),
			commands.RunCommand(),
			commands.StatusCommand(),
			commands.ApproveCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to configuration file",
				Value:   ".bridge/config.yaml",
				EnvVars: []string{"BRIDGE_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				EnvVars: []string{"BRIDGE_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (text, json)",
				Value:   "text",
				EnvVars: []string{"BRIDGE_OUTPUT"},
			},
		},
	}

	return app
}
