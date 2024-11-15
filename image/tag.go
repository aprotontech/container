package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/types"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func TagImageCommand(cmd *cobra.Command, args []string) {
	lp, err := Repository()
	utils.Assert(err)

	org, err := name.ParseReference(args[0])
	utils.Assert(err)

	dst, err := name.ParseReference(args[1])
	utils.Assert(err)

	ii, err := lp.ImageIndex()
	utils.Assert(err)

	imf, err := ii.IndexManifest()
	utils.Assert(err)

	for _, img := range imf.Manifests {
		if img.MediaType == types.DockerManifestSchema2 || img.MediaType == types.OCIManifestSchema1 {
			if name, ok := img.Annotations[oci.AnnotationRefName]; ok && name == org.Name() {
				i, err := lp.Image(img.Digest)
				utils.Assert(err)
				err = lp.ReplaceImage(i, match.Name(dst.Name()), layout.WithAnnotations(map[string]string{
					oci.AnnotationRefName: dst.Name(),
				}))
				utils.Assert(err)
				break
			}
		}
	}
}
