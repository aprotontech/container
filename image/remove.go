package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func RemoveImageCommand(cmd *cobra.Command, args []string) {
	lp, err := Repository()
	utils.Assert(err)

	ref, err := name.ParseReference(args[0])
	utils.Assert(err)

	err = lp.RemoveDescriptors(match.Name(ref.Name()))
	utils.Assert(err)
}
