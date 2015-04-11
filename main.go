package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func rootPath() string {
	path := os.Getenv("BUNDLE_PATH")
	if path == "" {
		path = "$HOME/bundle"
	} else if path[:2] == "~/" {
		path = "$HOME/" + path[2:]
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return os.ExpandEnv(path)
}

func dirContent(dir string) ([]os.FileInfo, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	return f.Readdir(-1)
}

func getBundles(dir string) (fileInfos []os.FileInfo, err error) {
	fis, err := dirContent(dir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if fi.IsDir() && fi.Name() != "usr" {
			fileInfos = append(fileInfos, fi)
		}
	}
	return
}

func IsSymlink(mode os.FileMode) bool {
	return mode&os.ModeSymlink == os.ModeSymlink
}

func getSymlinkFiles(dir string) (fileInfos []os.FileInfo, err error) {
	fis, err := dirContent(dir)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		if !IsSymlink(fi.Mode()) {
			continue
		}
		fileInfos = append(fileInfos, fi)
	}
	return
}

func deleteDeadLink(dir string) error {
	fis, err := getSymlinkFiles(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		s := dir + "/" + fi.Name()
		_, e := os.Stat(s)
		if os.IsNotExist(e) {
			os.Remove(s)
		}
	}
	return nil
}

func getLinkMap(dir string) (linkMap map[string]string, err error) {
	fis, err := getSymlinkFiles(dir)
	if err != nil {
		return
	}
	linkMap = make(map[string]string)
	for _, fi := range fis {
		var d string
		s := dir + "/" + fi.Name()
		d, _ = os.Readlink(s)
		if d[0] != '/' {
			d = filepath.Clean(dir + "/" + d)
		}
		linkMap[d] = s
	}
	return
}

func link(srcDir, destDir string) (err error) {
	_, err = os.Lstat(srcDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(srcDir, 0755)
		if err != nil {
			return
		}
	}
	if err != nil {
		return
	}

	lm, err := getLinkMap(srcDir)
	if err != nil {
		return
	}
	fis, err := dirContent(destDir)
	if err != nil {
		return
	}
	for _, fi := range fis {
		d := filepath.Clean(destDir + "/" + fi.Name())
		s, exist := lm[d]
		if !exist {
			if filepath.Base(s) == fi.Name() {
				p, _ := os.Readlink(s)
				fmt.Printf("WARNING: %s link to %s, can not relink to %s\n", s, p, d)
				continue
			}
			src := srcDir + "/" + fi.Name()
			dest := destDir + "/" + fi.Name()
			rel, e := filepath.Rel(srcDir, dest)
			if e == nil {
				dest = rel
			}
			fmt.Printf("%s -> %s\n", src, dest)
			err = os.Symlink(dest, src)
			if err != nil {
				return
			}
		}
	}
	return
}

func main() {
	rp := rootPath()

	deleteDeadLink(rp + "/usr/bin")

	fis, err := getBundles(rp)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	for _, fi := range fis {
		err := link(rp+"/usr/bin", rp+"/"+fi.Name()+"/bin")
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
	}
}
