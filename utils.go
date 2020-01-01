package sdkman

import (
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

// MkdirIfNotExist creates "candidates" and "archives" directory
func MkdirIfNotExist(root string) error {
	candDir := path.Join(root, "candidates")
	arcDir := path.Join(root, "archives")
	e := os.MkdirAll(candDir, os.ModeDir|os.ModePerm)
	if e != nil {
		return e
	}
	return os.MkdirAll(arcDir, os.ModeDir|os.ModePerm)
}

func platform() string {
	platform := runtime.GOOS

	if platform == "windows" {
		platform = "mingw"
		if runtime.GOARCH == "amd64" {
			platform += "64"
		}
	}
	if runtime.GOARCH == "386" || runtime.GOARCH == "amd64p32" {
		platform += "32"
	}
	return platform
}

func pager(pages io.ReadCloser) {
	pager := os.Getenv("PAGER")
	p := platform()

	if pager == "" {
		if strings.HasPrefix(platform(), "mingw") {
			pager = "more"
		} else {
			pager = "less"
		}
	}
	c1 := exec.Command(pager)
	c1.Stdin = pages
	c1.Stdout = os.Stdout
	_ = c1.Start()
	_ = c1.Wait()
	defer pages.Close()
}
