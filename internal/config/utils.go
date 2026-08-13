package config

import (
	"io/fs"
	"net/url"
	"os"
	"strings"
)

func isURL(link string) bool {
	link = strings.TrimSpace(link)
	u, e := url.Parse(link)
	return link != "" && e == nil && u.Host != ""
}

func isPath(path string) bool {
	path = strings.TrimSpace(path)
	return path != "" && fs.ValidPath(path)
}

func getTmpFilePath(appname, tmppath string) (_ string, e error) {
	var fd *os.File
	if fd, e = os.CreateTemp(tmppath, appname+"_dynamic*.yaml"); e != nil {
		return
	}

	return fd.Name(), nil
}
