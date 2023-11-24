package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func copyFile(src string, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// if it's a dir, just re-create it in build/
	if info.IsDir() {
		err := os.Mkdir(dest, info.Mode())
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		return nil
	}

	// open input
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// create output
	fh, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fh.Close()

	// match file permissions
	err = fh.Chmod(info.Mode())
	if err != nil {
		return err
	}

	// copy content
	_, err = io.Copy(fh, in)
	return err
}

func copyDirRecursively(src string, dst string) error {
	defer measure("copyDirRecursively")()

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		outpath := dst + strings.TrimPrefix(path, src)
		return copyFile(path, outpath)
	})
}
