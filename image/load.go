package image

import (
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func LoadImageCommand(cmd *cobra.Command, args []string) {
	inputFile := ""
	if flag := cmd.Flag("input"); flag != nil {
		inputFile = flag.Value.String()
	}

	tag, err := findTagNameInFile(inputFile)
	utils.Assert(err)

	ref, err := name.ParseReference(tag)
	utils.Assert(err)

	img, err := tarball.Image(func() (io.ReadCloser, error) {
		return os.Open(inputFile)
	}, nil)
	utils.Assert(err)

	lp, err := Repository()
	utils.Assert(err)
	err = lp.ReplaceImage(img, match.Name(ref.Name()), layout.WithAnnotations(map[string]string{
		oci.AnnotationRefName: ref.Name(),
	}))
	utils.Assert(err)

}
