package image

import (
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func SaveImageCommand(cmd *cobra.Command, args []string) {
	output := os.Stdout
	if flag := cmd.Flag("output"); flag != nil && flag.Value.String() != "" {
		var err error
		output, err = os.Create(flag.Value.String())
		utils.Assert(err)
		defer output.Close()
	}

	lp, err := Repository()
	utils.Assert(err)

	ii, err := lp.ImageIndex()
	utils.Assert(err)

	imf, err := ii.IndexManifest()
	utils.Assert(err)

	for _, tag := range args {
		logrus.Infof("name=%s", tag)
		ref, err := name.ParseReference(tag)
		utils.Assert(err)

		for _, img := range imf.Manifests {
			if img.MediaType == types.DockerManifestSchema2 || img.MediaType == types.OCIManifestSchema1 {
				if name, ok := img.Annotations[oci.AnnotationRefName]; ok && name == ref.Name() {
					logrus.Infof("found=%s", name)
					i, err := lp.Image(img.Digest)
					utils.Assert(err)
					utils.Assert(tarball.Write(ref, i, output))
					break
				}
			}
		}

	}

	utils.Assert(err)

}
