package run

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/pflag"
)

type Cli struct {
	Command     string
	ComposePath string
	TemplateId  string
	ServiceName string
	ProjectName string
	ProjectPath string
	SshTarget   string
}

// ParseCli maintained for backwards-compatible test (was parseCli returning help when no args)
func ParseCli() Cli { return parseCliArgs(os.Args[1:]) }

func parseCliArgs(args []string) Cli {
	var cli Cli
	fs := pflag.NewFlagSet("topo", pflag.ContinueOnError)
	_ = fs.Parse(args) // ignore flag errors for parity
	pos := fs.Args()
	if len(pos) == 0 {
		cli.Command = "help"
		return cli
	}
	cli.Command = pos[0]
	switch cli.Command {
	case "add-service":
		if len(pos) > 2 {
			cli.ComposePath = pos[1]
			cli.TemplateId = pos[2]
			if len(pos) > 3 {
				cli.ServiceName = pos[3]
			} else {
				cli.ServiceName = cli.TemplateId
			}
		}
	case "remove-service":
		if len(pos) > 2 {
			cli.ComposePath = pos[1]
			cli.ServiceName = pos[2]
		}
	case "get-project":
		if len(pos) > 1 {
			cli.ComposePath = pos[1]
		}
	case "init-project":
		if len(pos) > 2 {
			cli.ProjectPath = pos[1]
			cli.ProjectName = pos[2]
			if len(pos) > 3 {
				cli.SshTarget = pos[3]
			}
		}
	case "generate-makefile":
		if len(pos) > 1 {
			cli.ComposePath = pos[1]
			if len(pos) > 2 {
				cli.SshTarget = pos[2]
			}
		}
	case "get-containers-info":
		if len(pos) > 1 {
			cli.SshTarget = pos[1]
		}
	}
	return cli
}

func (c Cli) validate() error {
	req := func(ok bool, msg string) error {
		if !ok {
			return errors.New(msg)
		}
		return nil
	}
	switch c.Command {
	case "add-service":
		return req(c.ComposePath != "" && c.TemplateId != "", "missing arguments for 'add-service'")
	case "remove-service":
		return req(c.ComposePath != "" && c.ServiceName != "", "missing arguments for 'remove-service'")
	case "get-project":
		return req(c.ComposePath != "", "missing compose path for 'get-project'")
	case "init-project":
		return req(c.ProjectPath != "" && c.ProjectName != "", "missing arguments for 'init-project'")
	case "generate-makefile":
		return req(c.ComposePath != "", "missing compose path for 'generate-makefile'")
	}
	return nil
}

func Execute(args []string, stdout, stderr io.Writer) error {
	cli := parseCliArgs(args)
	if cli.Command == "help" || cli.Command == "" {
		core.PrintHelp()
		return nil
	}
	if err := cli.validate(); err != nil {
		return err
	}
	switch cli.Command {
	case "list-templates":
		return core.ListTemplates()
	case "get-config-metadata":
		return core.GetConfigMetadata()
	case "version":
		core.PrintVersion()
		return nil
	case "get-project":
		return core.GetProject(cli.ComposePath)
	case "init-project":
		return core.RunInitProject(cli.ProjectPath, cli.ProjectName, cli.SshTarget)
	case "add-service":
		return core.RunAddService(cli.ComposePath, cli.TemplateId, cli.ServiceName, core.CloneProject)
	case "remove-service":
		return core.RunRemoveService(cli.ComposePath, cli.ServiceName)
	case "generate-makefile":
		return core.GenerateMakefile(cli.ComposePath, cli.SshTarget)
	case "get-containers-info":
		return core.GetContainersInfo(cli.SshTarget)
	default:
		return fmt.Errorf("unknown command: %s", cli.Command)
	}
}
