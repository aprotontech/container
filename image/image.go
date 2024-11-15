package image

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/lithammer/shortuuid"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"

	"aproton.tech/container/utils"
)

const SandboxPath = "var/sandbox"
const RepositoryPath = "var/repositories"
const ImageLowerPath = "var/overlay/lower"
const TempPath = "var/tmp"

func ImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "image",
		Aliases: []string{"images", "img"},
		Short:   "image commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list images",
		Aliases: []string{"ls"},
		Short:   "list images",
		Run:     ListImageCommand,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "pull image",
		Short: "pull image",
		Args:  cobra.ExactArgs(1),
		Run:   PullImageCommand,
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "remove image",
		Short:   "remove image",
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		Run:     RemoveImageCommand,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "tag image new_name",
		Short: "rename image",
		Args:  cobra.ExactArgs(2),
		Run:   TagImageCommand,
	})

	load := &cobra.Command{
		Use:   "load image -i file.tar.gz",
		Short: "load image",
		Args:  cobra.NoArgs,
		Run:   LoadImageCommand,
	}
	load.Flags().StringP("input", "i", "", "--input=file.tar.gz")

	save := &cobra.Command{
		Use:   "save [OPTIONS] IMAGE [IMAGE...]\nSave one or more images to a tar archive (streamed to STDOUT by default)",
		Short: "save image",
		Args:  cobra.MinimumNArgs(1),
		Run:   SaveImageCommand,
	}
	save.Flags().StringP("output", "o", "", "Write to a file, instead of STDOUT")

	cmd.AddCommand(load)
	cmd.AddCommand(save)

	return cmd
}

func GetImage(image string, forcePull bool) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	utils.Assert(err)

	lp, err := Repository()
	if err != nil {
		return nil, err
	}

	if !forcePull {
		// check image is exists
		ii, err := lp.ImageIndex()
		if err != nil {
			return nil, err
		}

		imf, err := ii.IndexManifest()
		if err != nil {
			return nil, err
		}

		for _, img := range imf.Manifests {
			if img.MediaType == types.DockerManifestSchema2 || img.MediaType == types.OCIManifestSchema1 {
				if name, ok := img.Annotations[oci.AnnotationRefName]; ok && name == ref.Name() {
					return lp.Image(img.Digest)
				}
			}
		}
	}

	return doPullImage(ref)
}

func BuildSandbox(img v1.Image, sboxID string) string {
	newsdx := path.Join(SandboxPath, sboxID)

	rc := mutate.Extract(img)
	defer rc.Close()

	utils.Assert(os.MkdirAll(newsdx, 0755))

	if err := utils.Untar(rc, newsdx); err != nil {
		os.RemoveAll(newsdx)
		utils.Assert(err)
	}

	return newsdx
}

func FlatImageFiles(img v1.Image) []string {
	hash, err := img.Digest()
	utils.Assert(err)

	path := filepath.Join(ImageLowerPath, hash.Hex)
	_, err = os.Stat(path)

	if err == nil {
		return []string{path}
	}

	if errors.Is(err, os.ErrNotExist) {
		tmp := filepath.Join(TempPath, shortuuid.New())
		utils.Assert(os.MkdirAll(tmp, 0755))

		rc := mutate.Extract(img)
		defer rc.Close()

		if err := utils.Untar(rc, tmp); err != nil {
			os.RemoveAll(tmp)
			utils.Assert(err)
		}

		utils.Assert(os.MkdirAll(ImageLowerPath, 0755))
		if err := os.Rename(tmp, path); err != nil {
			// file path is exists
			_, err := os.Stat(path)
			utils.Assert(err)
		}
	}

	return []string{path}
}

func GetImageConfig(img v1.Image) (*v1.Config, error) {
	config, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	return config.Config.DeepCopy(), nil
}

func Repository() (layout.Path, error) {
	lp, err := layout.FromPath(RepositoryPath)

	if err != nil {
		lp, err = layout.Write(RepositoryPath, empty.Index)
		if err != nil {
			return "", err
		}
	}

	return lp, nil
}

func findTagNameInFile(tarFile string) (string, error) {
	file, err := os.Open(tarFile)
	if err != nil {
		return "", err
	}

	defer file.Close()

	tf := tar.NewReader(file)
	for hdr, err := tf.Next(); err == nil; hdr, err = tf.Next() {
		if hdr.Name == "manifest.json" {
			manifest := tarball.Manifest{}
			json.NewDecoder(tf).Decode(&manifest)
			if len(manifest) == 0 {
				return "", errors.New("manifest.json format error")
			}
			if len(manifest[0].RepoTags) == 0 {
				name := strings.TrimSuffix(path.Base(tarFile), path.Ext(tarFile))
				return name, nil
			}

			return manifest[0].RepoTags[0], nil
		}
	}

	return "", err
}
