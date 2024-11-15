package image

import (
	"errors"
	"os"

	"github.com/dustin/go-humanize"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/olekukonko/tablewriter"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/util/parsers"

	"aproton.tech/container/utils"
)

func ListImageCommand(cmd *cobra.Command, args []string) {
	lp, err := Repository()
	utils.Assert(err)

	ii, err := lp.ImageIndex()
	utils.Assert(err)

	imf, err := ii.IndexManifest()
	utils.Assert(err)

	table := newImageListTableRender()
	for _, img := range imf.Manifests {
		if row, err := newImageListRow(ii, &img); err == nil {
			table.Append(row)
		}
	}
	table.Render()
}

func newImageListTableRender() *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"REPOSITORY", "TAG", "IMAGE ID", "SIZE"})
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

func newImageListRow(ii v1.ImageIndex, img *v1.Descriptor) ([]string, error) {
	if img.MediaType == types.DockerManifestSchema2 || img.MediaType == types.OCIManifestSchema1 {
		if name, ok := img.Annotations[oci.AnnotationRefName]; ok {
			repoToPull, tag, _, err := parsers.ParseImageName(name)
			if err != nil {
				return nil, err
			}

			real, err := ii.Image(img.Digest)
			if err != nil {
				return nil, err
			}

			layers, err := real.Layers()
			if err != nil {
				return nil, err
			}

			size := int64(0)
			for _, layer := range layers {
				s, err := layer.Size()
				if err == nil {
					size += s
				}
			}

			return []string{
				repoToPull,
				tag,
				img.Digest.String()[7:19],
				humanize.Bytes(uint64(size)),
			}, nil
		}
	}

	return nil, errors.New("not image")
}
