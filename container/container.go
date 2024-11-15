package container

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

const ReExecRunCommand = "inner-container-run"
const ContainerMetaFile = "var/container.json"

type ContainerMeta struct {
	Name        string    `json:"name"`
	ProcessID   int       `json:"processId"`
	ContainerID string    `json:"containerId"`
	Image       string    `json:"image"`
	Command     string    `json:"command"`
	Created     time.Time `json:"created"`
	Status      string    `json:"status"`
	Ports       string    `json:"ports"`
	Sandbox     string    `json:"sandbox"`
	Overlay     *Overlay  `json:"overlay"`
}

type Overlay struct {
	Working    string `json:"working"`
	Upper      string `json:"upper"`
	MountPoint string `json:"mountPoint"`
}

func ContainerCommands() []*cobra.Command {
	cmdlist := getCommands()

	containercmd := &cobra.Command{
		Use:   "container",
		Short: "container commands",
	}
	for _, cmd := range getCommands() {
		containercmd.AddCommand(cmd)
	}

	return append(cmdlist, containercmd)
}

func getCommands() []*cobra.Command {
	run := &cobra.Command{
		Use:   "run container",
		Short: "start and run a container",
		Args:  cobra.MinimumNArgs(1),
		Run:   ContainerRunCommand,
	}

	run.Flags().BoolP("interactive", "i", false, "Keep STDIN open even if not attached")
	run.Flags().BoolP("tty", "t", false, "Allocate a pseudo-TTY")
	run.Flags().BoolP("detach", "d", false, "Run container in background and print container ID")
	run.Flags().BoolP("rm", "", false, "Automatically remove the container when it exits")
	run.Flags().StringP("memory", "m", "", "Memory limit")

	list := &cobra.Command{
		Use:     "ps",
		Short:   "list containers",
		Aliases: []string{"list", "ls"},
		Run:     ContainerListCommand,
	}

	stop := &cobra.Command{
		Use:   "stop",
		Short: "stop containers",
		Args:  cobra.MinimumNArgs(1),
		Run:   ContainerStopCommand,
	}

	remove := &cobra.Command{
		Use:     "remove",
		Short:   "remove containers",
		Args:    cobra.MinimumNArgs(1),
		Aliases: []string{"rm"},
		Run:     ContainerRemoveCommand,
	}

	return []*cobra.Command{run, list, stop, remove}
}

func getContainerMetas() ([]*ContainerMeta, error) {
	content, err := os.ReadFile(ContainerMetaFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*ContainerMeta{}, nil
		}
	}

	var containers []*ContainerMeta
	utils.Assert(json.Unmarshal(content, &containers))
	return containers, nil
}

func getContainerMetasMap() (map[string]*ContainerMeta, error) {
	containers, err := getContainerMetas()
	if err != nil {
		return nil, err
	}

	cmap := map[string]*ContainerMeta{}
	for idx, cnt := range containers {
		cmap[cnt.ContainerID] = containers[idx]
		cmap[cnt.Name] = containers[idx]
	}

	return cmap, nil
}

func removeContainerMeta(remove *ContainerMeta) {
	containers, err := getContainerMetas()
	utils.Assert(err)

	left := []*ContainerMeta{}
	for _, cnt := range containers {
		if cnt.ContainerID != remove.ContainerID {
			left = append(left, cnt)
		}
	}

	content, err := json.Marshal(left)
	utils.Assert(err)
	utils.Assert(os.WriteFile(ContainerMetaFile, content, 0644))
}

func appendContainerMeta(cnt *ContainerMeta) {
	containers, err := getContainerMetas()
	utils.Assert(err)

	containers = append(containers, cnt)

	content, err := json.Marshal(containers)
	utils.Assert(err)
	utils.Assert(os.WriteFile(ContainerMetaFile, content, 0644))
}

func updateContainerMeta(update *ContainerMeta) {
	containers, err := getContainerMetas()
	utils.Assert(err)

	for _, cnt := range containers {
		if cnt.ContainerID == update.ContainerID {
			*cnt = *update
			break
		}
	}

	content, err := json.Marshal(containers)
	utils.Assert(err)
	utils.Assert(os.WriteFile(ContainerMetaFile, content, 0644))
}
