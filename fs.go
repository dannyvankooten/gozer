package main

import (
	"errors"
	"github.com/fsnotify/fsnotify"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func watchDirs(dirs []string, cb func()) {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("error creating fsnotify watcher")
		return
	}
	defer watcher.Close()

	for _, p := range dirs {
		if err := filepath.WalkDir(p, func(f string, d fs.DirEntry, err error) error {
			if !d.IsDir() {
				return nil
			}

			return watcher.Add(f)
		}); err != nil {
			log.Fatal("error adding directory to watcher: %s", err)
		}
	}

	// block thread indefinitely
	triggered := time.Now()
	for event := range watcher.Events {
		if event.Has(fsnotify.Write) && time.Since(triggered) > 1*time.Second {
			time.Sleep(100 * time.Millisecond)
			triggered = time.Now()
			cb()
		}
	}
}

func copyFile(src string, d fs.DirEntry, dest string) error {
	// if it's a dir, just re-create it in build/
	if d.IsDir() {
		err := os.MkdirAll(dest, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		return nil
	}

	// open source file
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// create dest file
	fh, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fh.Close()

	// copy src content into dest content
	_, err = io.Copy(fh, in)
	return err
}

func copyDirRecursively(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		outpath := dst + strings.TrimPrefix(path, src)
		return copyFile(path, d, outpath)
	})
}
