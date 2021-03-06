package main

/*
	Definition for the main entry command. This defined the "human"
	interfaces for `run` and `run-steps`
*/

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	rhjobspec "github.com/coreos/entrypoint/spec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const cosaContainerDir = "/usr/lib/coreos-assembler"

var (
	version = "devel"

	// cosaDir is the installed location of COSA. This defaults to
	// cosaContainerDir and is set via `-ldflags` at build time.
	cosaDir string

	// spec is an RHCOS spec. It is anticipated that this will be
	// changed in the future.
	spec     rhjobspec.JobSpec
	specFile string

	// entryEnvars are set for command execution
	entryEnvVars []string

	// shellCmd is the default command to execute commands.
	shellCmd = []string{"/bin/bash", "-x"}

	cmdRoot = &cobra.Command{
		Use:   "entry [command]",
		Short: "COSA entrypoint",
		Long: `Entrypoint for CoreOS Assemlber
Wrapper for COSA commands and templates`,
		PersistentPreRun: preRun,
	}

	cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Print the version number and exit.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("entry/%s version %s\n",
				cmd.Root().Name(), version)
		},
	}

	cmdSingle = &cobra.Command{
		Use:   "run",
		Short: "Run the commands and bail",
		Args:  cobra.MinimumNArgs(1),
		Run:   runSingle,
	}

	cmdSteps = &cobra.Command{
		Use:          "run-scripts",
		Short:        "Run Steps from [file]",
		Args:         cobra.MinimumNArgs(1),
		RunE:         runScripts,
		SilenceUsage: true,
	}
)

func init() {
	if cosaDir == "" {
		path, err := os.Getwd()
		if err != nil {
			cosaDir = cosaContainerDir
		} else {
			cosaDir = filepath.Dir(path)
		}
	}

	entryEnvVars = os.Environ()

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	newPath := fmt.Sprintf("%s:%s", cosaDir, os.Getenv("PATH"))
	os.Setenv("PATH", newPath)

	// cmdRoot options
	cmdRoot.PersistentFlags().StringVarP(&specFile, "spec", "s", "", "location of the spec")
	cmdRoot.AddCommand(cmdVersion)
	cmdRoot.AddCommand(cmdSingle)

	// cmdStep options
	cmdRoot.AddCommand(cmdSteps)
	cmdSteps.Flags().StringVarP(&specFile, "spec", "s", "", "location of the spec")
	cmdSteps.Flags().StringArrayVarP(&shellCmd, "shell", "S", shellCmd, "shellcommand to execute")
}

func main() {
	log.Infof("CoreOS-Assembler Entrypoint, %s", version)
	if err := cmdRoot.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

// runScripts reads ARGs as files and executes the rendered templates.
func runScripts(c *cobra.Command, args []string) error {
	if err := spec.RendererExecuter(ctx, entryEnvVars, args...); err != nil {
		log.Fatalf("Failed to execute scripts: %v", err)
	}
	log.Infof("Execution complete")
	return nil
}

// runSingle renders args as templates and executes the command.
func runSingle(c *cobra.Command, args []string) {
	x, err := spec.ExecuteTemplateFromString(args...)
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, x[0], x[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	log.Infof("Done")
}

// preRun processes the spec file.
func preRun(c *cobra.Command, args []string) {
	if specFile == "" {
		return
	}

	ns, err := rhjobspec.JobSpecFromFile(specFile)
	if err != nil {
		log.WithFields(log.Fields{"input file": specFile, "error": err}).Fatal(
			"Failed reading file")
	}
	spec = *ns
}
