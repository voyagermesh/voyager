package ioutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/appscode/go/log"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	maxFileNameLength = 255
	maxPathLength     = 4096
)

// Adapted from: https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/util/atomic_writer.go

// AtomicWriter handles atomically projecting content for a set of files into
// a target directory.
//
// Note:
//
// 1. AtomicWriter reserves the set of pathnames starting with `..`.
// 2. AtomicWriter offers no concurrency guarantees and must be synchronized
//    by the caller.
//
type AtomicWriter struct {
	targetDir string
}

type FileProjection struct {
	Data []byte
	Mode int32
}

// NewAtomicWriter creates a new AtomicWriter configured to write to the given
// target directory, or returns an error if the target directory does not exist.
func NewAtomicWriter(targetDir string) (*AtomicWriter, error) {
	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		return nil, err
	}

	return &AtomicWriter{targetDir: targetDir}, nil
}

func (w *AtomicWriter) Write(payload map[string]FileProjection) (bool, error) {
	cleanPayload, err := validatePayload(payload)
	if err != nil {
		log.Errorf("invalid payload: %v", err)
		return false, err
	}

	pathsToRemove, err := w.pathsToRemove(cleanPayload)
	if err != nil {
		log.Errorf("error determining user-visible files to remove: %v", err)
		return false, err
	}

	if should, err := w.shouldWritePayload(cleanPayload); err != nil {
		log.Errorf("error determining whether payload should be written to disk: %v", err)
		return false, err
	} else if !should && len(pathsToRemove) == 0 {
		log.V(4).Infof("no update required for target directory %v", w.targetDir)
		return false, nil
	} else {
		log.V(4).Infof("write required for target directory %v", w.targetDir)
	}

	if err = w.writePayloadToDir(cleanPayload, w.targetDir); err != nil {
		log.Errorf("error writing payload to ts data directory %s: %v", w.targetDir, err)
		return false, err
	} else {
		log.V(4).Infof("performed write of new data to ts data directory: %s", w.targetDir)
	}

	if err = w.removeUserVisiblePaths(pathsToRemove); err != nil {
		log.Errorf("error removing old visible symlinks: %v", err)
		return false, err
	}

	return true, nil
}

// validatePayload returns an error if any path in the payload  returns a copy of the payload with the paths cleaned.
func validatePayload(payload map[string]FileProjection) (map[string]FileProjection, error) {
	cleanPayload := make(map[string]FileProjection)
	for k, content := range payload {
		if err := validatePath(k); err != nil {
			return nil, err
		}

		cleanPayload[path.Clean(k)] = content
	}

	return cleanPayload, nil
}

// validatePath validates a single path, returning an error if the path is
// invalid.  paths may not:
//
// 1. be absolute
// 2. contain '..' as an element
// 3. start with '..'
// 4. contain filenames larger than 255 characters
// 5. be longer than 4096 characters
func validatePath(targetPath string) error {
	// TODO: somehow unify this with the similar api validation,
	// validateVolumeSourcePath; the error semantics are just different enough
	// from this that it was time-prohibitive trying to find the right
	// refactoring to re-use.
	if targetPath == "" {
		return fmt.Errorf("invalid path: must not be empty: %q", targetPath)
	}
	if path.IsAbs(targetPath) {
		return fmt.Errorf("invalid path: must be relative path: %s", targetPath)
	}

	if len(targetPath) > maxPathLength {
		return fmt.Errorf("invalid path: must be less than %d characters", maxPathLength)
	}

	items := strings.Split(targetPath, string(os.PathSeparator))
	for _, item := range items {
		if item == ".." {
			return fmt.Errorf("invalid path: must not contain '..': %s", targetPath)
		}
		if len(item) > maxFileNameLength {
			return fmt.Errorf("invalid path: filenames must be less than %d characters", maxFileNameLength)
		}
	}
	if strings.HasPrefix(items[0], "..") && len(items[0]) > 2 {
		return fmt.Errorf("invalid path: must not start with '..': %s", targetPath)
	}

	return nil
}

// shouldWritePayload returns whether the payload should be written to disk.
func (w *AtomicWriter) shouldWritePayload(payload map[string]FileProjection) (bool, error) {
	for userVisiblePath, fileProjection := range payload {
		shouldWrite, err := w.shouldWriteFile(path.Join(w.targetDir, userVisiblePath), fileProjection.Data)
		if err != nil {
			return false, err
		}

		if shouldWrite {
			return true, nil
		}
	}

	return false, nil
}

// shouldWriteFile returns whether a new version of a file should be written to disk.
func (w *AtomicWriter) shouldWriteFile(path string, content []byte) (bool, error) {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return true, nil
	}

	contentOnFs, err := ioutil.ReadFile(path)
	if err != nil {
		return false, err
	}

	return (bytes.Compare(content, contentOnFs) != 0), nil
}

// writePayloadToDir writes the given payload to the given directory.  The
// directory must exist.
func (w *AtomicWriter) writePayloadToDir(payload map[string]FileProjection, dir string) error {
	for userVisiblePath, fileProjection := range payload {
		content := fileProjection.Data

		if fileProjection.Mode <= 0 {
			// Setting Default Mode
			fileProjection.Mode = 0777
		}
		mode := os.FileMode(fileProjection.Mode)
		fullPath := path.Join(dir, userVisiblePath)
		baseDir, _ := filepath.Split(fullPath)
		err := os.MkdirAll(baseDir, os.ModePerm)
		if err != nil {
			log.Errorf("unable to create directory %s: %v", baseDir, err)
			return err
		}

		err = ioutil.WriteFile(fullPath, content, mode)
		if err != nil {
			log.Errorf("unable to write file %s with mode %v: %v", fullPath, mode, err)
			return err
		}
		// Chmod is needed because ioutil.WriteFile() ends up calling
		// open(2) to create the file, so the final mode used is "mode &
		// ~umask". But we want to make sure the specified mode is used
		// in the file no matter what the umask is.
		err = os.Chmod(fullPath, mode)
		if err != nil {
			log.Errorf("unable to write file %s with mode %v: %v", fullPath, mode, err)
		}
	}
	return nil
}

func (w *AtomicWriter) pathsToRemove(payload map[string]FileProjection) (sets.String, error) {
	paths := sets.NewString()
	visitor := func(path string, info os.FileInfo, err error) error {
		if path == w.targetDir {
			return nil
		}

		relativePath := strings.TrimPrefix(path, w.targetDir)
		if runtime.GOOS == "windows" {
			relativePath = strings.TrimPrefix(relativePath, "\\")
		} else {
			relativePath = strings.TrimPrefix(relativePath, "/")
		}
		if strings.HasPrefix(relativePath, "..") {
			return nil
		}

		paths.Insert(relativePath)
		return nil
	}

	err := filepath.Walk(w.targetDir, visitor)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	log.V(5).Infof("%s: current paths:   %+v", w.targetDir, paths.List())

	newPaths := sets.NewString()
	for file := range payload {
		// add all subpaths for the payload to the set of new paths
		// to avoid attempting to remove non-empty dirs
		for subPath := file; subPath != ""; {
			newPaths.Insert(subPath)
			subPath, _ = filepath.Split(subPath)
			subPath = strings.TrimSuffix(subPath, "/")
		}
	}
	log.V(5).Infof("%s: new paths:       %+v", w.targetDir, newPaths.List())

	result := paths.Difference(newPaths)
	log.V(5).Infof("%s: paths to remove: %+v", w.targetDir, result)
	return result, nil
}

// removeUserVisiblePaths removes the set of paths from the user-visible
// portion of the writer's target directory.
func (w *AtomicWriter) removeUserVisiblePaths(paths sets.String) error {
	orderedPaths := paths.List()
	for ii := len(orderedPaths) - 1; ii >= 0; ii-- {
		if err := os.Remove(path.Join(w.targetDir, orderedPaths[ii])); err != nil {
			log.Errorf("error pruning old user-visible path %s: %v", orderedPaths[ii], err)
			return err
		}
	}

	return nil
}
