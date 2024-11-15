package utils

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func Untar(r io.Reader, path string) error {
	dirPermissionRevertList := map[string]fs.FileMode{}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		to := filepath.Join(path, hdr.Name)

		if hdr.Typeflag == tar.TypeDir {
			if (hdr.FileInfo().Mode().Perm() & 0200) == 0 {
				orgMod := hdr.Mode
				hdr.Mode = hdr.Mode | 0200
				logrus.Debugf("path(%s) change permission (%s) --> (%s)", to, fs.FileMode(orgMod).String(), hdr.FileInfo().Mode().String())

				dirPermissionRevertList[to] = fs.FileMode(orgMod)
			}
		}

		canRetry, err := untarFile(to, hdr, tr)
		if err != nil {
			if !canRetry {
				return err
			}

			if err := os.RemoveAll(to); err != nil {
				return err
			}
			if _, err := untarFile(to, hdr, tr); err != nil {
				return err
			}
		}
	}

	for dir, mod := range dirPermissionRevertList {
		err := os.Chmod(dir, mod)
		logrus.Debugf("path(%s) revert permission to (%s), err=%v", dir, mod.String(), err)
	}

	return nil
}

func untarFile(to string, hdr *tar.Header, r *tar.Reader) (bool, error) {
	f := hdr.FileInfo()

	getRealLinkName := func(linkname string) string {
		base := to[:(len(to) - len(hdr.Name))]
		return path.Join(base, linkname)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return true, os.MkdirAll(to, f.Mode())
	case tar.TypeReg, tar.TypeChar, tar.TypeBlock, tar.TypeFifo, tar.TypeGNUSparse:
		f, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_RDWR, f.Mode())
		if err != nil {
			return true, err
		}

		defer f.Close()
		if _, err := io.Copy(f, r); err != nil {
			return false, err
		}
		return false, nil
	case tar.TypeSymlink:
		return true, os.Symlink(hdr.Linkname, to)
	case tar.TypeLink:
		return true, os.Link(getRealLinkName(hdr.Linkname), to)
	case tar.TypeXGlobalHeader:
		return false, nil
	default:
		return false, fmt.Errorf("%s: unknown type flag: %c", hdr.Name, hdr.Typeflag)
	}
}
