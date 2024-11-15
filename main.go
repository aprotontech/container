package main

import (
	"fmt"
	"os"

	"github.com/moby/moby/pkg/reexec"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"aproton.tech/container/container"
	"aproton.tech/container/image"
	"aproton.tech/container/utils"
)

func init() {
	logrus.SetFormatter(&utils.LogFormatter{})
	logrus.SetReportCaller(true)
	reexec.Register(container.ReExecRunCommand, func() {
		utils.SetSubProcessFlag()
		if err := container.Run(os.Args[1], os.Args[2]); err != nil {
			utils.Assert(err)
		}
		logrus.Infof("run finished")
	})
}

func main() {
	if reexec.Init() {
		os.Exit(0)
	}

	rootCmd := &cobra.Command{
		Use:               "container",
		Long:              `container`,
		Version:           "1.0",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	for _, cmd := range container.ContainerCommands() {
		rootCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(image.ImageCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
