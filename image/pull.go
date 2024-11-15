package image

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

func PullImageCommand(cmd *cobra.Command, args []string) {
	ref, err := name.ParseReference(args[0])
	utils.Assert(err)

	_, err = doPullImage(ref)

	utils.Assert(err)
}

func doPullImage(ref name.Reference) (v1.Image, error) {
	lp, err := Repository()
	utils.Assert(err)

	logrus.Infof("Current Platform: OS=%s, Architecture=%s", runtime.GOOS, runtime.GOARCH)

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	remoteOptions := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(remote.DefaultTransport),

		remote.WithPlatform(v1.Platform{
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
		}),
	}

	rmt, err := remote.Get(ref, remoteOptions...)
	utils.Assert(err)

	var img v1.Image
	if rmt.MediaType.IsIndex() {
		idx, err := rmt.ImageIndex()
		utils.Assert(err)
		if img, err = findMatchImage(idx); err != nil {
			utils.Assert(err)
		}
	} else {
		if img, err = rmt.Image(); err != nil {
			utils.Assert(err)
		}
	}

	// if image aready exists, will update it
	if err = lp.ReplaceImage(img, match.Name(ref.Name()), layout.WithAnnotations(map[string]string{
		oci.AnnotationRefName: ref.Name(),
	})); err != nil {
		utils.Assert(err)
	}

	return img, nil
}

func findMatchImage(idx v1.ImageIndex) (v1.Image, error) {
	manifests, err := partial.Manifests(idx)
	if err != nil {
		return nil, err
	}

	var matched partial.Describable
	for _, m := range manifests {
		// Keep the old descriptor (annotations and whatnot).
		desc, err := partial.Descriptor(m)
		if err != nil {
			return nil, err
		}

		// High-priority: platform/arch are matched
		// Middle-priority: arch matched
		if p := desc.Platform; p != nil {
			logrus.Infof("Image=%s, Platform: OS=%s, Architecture=%s", desc.Digest.String(), p.OS, p.Architecture)

			if p.Architecture == runtime.GOARCH && p.OS == runtime.GOOS {
				matched = m
				break
			} else if matched == nil && p.Architecture == runtime.GOARCH {
				matched = m
			}
		}
	}

	if matched != nil {
		if img, ok := matched.(v1.Image); ok {
			desc, _ := partial.Descriptor(matched)
			logrus.Infof("Selected best matched image=%s", desc.Digest.String())
			return img, nil
		}
		return nil, fmt.Errorf("found a matched index, but is not an image")
	}

	return nil, errors.New("not found matched image")
}
