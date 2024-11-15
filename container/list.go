package container

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func ContainerListCommand(cmd *cobra.Command, args []string) {
	_, err := os.Stat(ContainerMetaFile)
	if errors.Is(err, os.ErrNotExist) {
		return
	}

	content, err := os.ReadFile(ContainerMetaFile)
	utils.Assert(err)

	var containers []ContainerMeta
	utils.Assert(json.Unmarshal(content, &containers))

	table := newContainerListTableRender()
	for _, c := range containers {
		if !utils.IsProcessExists(c.ProcessID, c.Command) {
			c.Status = "Exited"
		}
		table.Append([]string{c.ContainerID, c.Image, c.Command, c.Created.Format("2006-01-02 15:04:05"), c.Status, c.Ports, c.Name})
	}
	table.Render()
}

func newContainerListTableRender() *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"CONTAINER ID", "IMAGE", "COMMAND", "CREATED", "STATUS", "PORTS", "NAMES"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	return table
}
