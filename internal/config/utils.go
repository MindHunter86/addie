package config

import (
	"io/fs"
	"net/url"
	"os"
)

func isURL(link string) bool {
	_, e := url.Parse(link)
	return e == nil
}

func isPath(path string) bool {
	return fs.ValidPath(path)
}

func getTmpFilePath(appname, tmppath string) (_ string, e error) {
	var fd *os.File
	if fd, e = os.CreateTemp(tmppath, appname+"_dynamic*.yaml"); e != nil {
		return
	}

	return fd.Name(), nil
}
