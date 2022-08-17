package probe

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/djherbis/times"
)

type FileProbe struct {
	BaseProbe

	// Path for file/directory existence and timestamp checks
	Path string `yaml:"path"`
	// Path for file/directory non-existence checks
	PathNotExists   string `yaml:"path_not_exists"`
	PathIsDirectory string `yaml:"path_is_directory"`
	PathIsFile      string `yaml:"path_is_file"`

	// Timestamp checks
	OlderThan    time.Duration `yaml:"older_than"`
	NotOlderThan time.Duration `yaml:"not_older_than"`

	AtimeOlderThan    time.Duration `yaml:"atime_older_than"`
	AtimeNotOlderThan time.Duration `yaml:"atime_not_older_than"`

	MtimeOlderThan    time.Duration `yaml:"mtime_older_than"`
	MtimeNotOlderThan time.Duration `yaml:"mtime_not_older_than"`

	CtimeOlderThan    time.Duration `yaml:"ctime_older_than"`
	CtimeNotOlderThan time.Duration `yaml:"ctime_not_older_than"`

	BtimeOlderThan    time.Duration `yaml:"btime_older_than"`
	BtimeNotOlderThan time.Duration `yaml:"btime_not_older_than"`
}

func (p *FileProbe) err(f string, args ...interface{}) error {
	return fmt.Errorf("file: "+f, args...)
}

func (p *FileProbe) olderThanNotOlderThanCheck(olderThan, notOlderThan time.Duration, t time.Time) error {
	if olderThan != 0 && time.Since(t) <= olderThan {
		return fmt.Errorf("is not older than %v", olderThan)
	} else if notOlderThan != 0 && time.Since(t) > notOlderThan {
		return fmt.Errorf("is older than %v", notOlderThan)
	}
	return nil
}

func (p *FileProbe) String() string {
	attrs := "path=" + p.Path
	return fmt.Sprintf("File(%s)", attrs)
}

func (p *FileProbe) runFileExistenceAndTimestampChecks() error {
	fi, err := times.Stat(p.Path)
	if err == nil {
		changeOrModTime := fi.ModTime()
		if fi.HasChangeTime() {
			ctime := fi.ChangeTime()
			if ctime.After(changeOrModTime) {
				changeOrModTime = ctime
			}
		}

		err = p.olderThanNotOlderThanCheck(p.OlderThan, p.NotOlderThan, changeOrModTime)

		if err == nil {
			err = p.olderThanNotOlderThanCheck(p.AtimeOlderThan, p.AtimeNotOlderThan, fi.AccessTime())
		}

		if err == nil {
			err = p.olderThanNotOlderThanCheck(p.MtimeOlderThan, p.MtimeNotOlderThan, fi.ModTime())
		}

		if err == nil && (p.CtimeOlderThan != 0 || p.CtimeNotOlderThan != 0) {
			if !fi.HasChangeTime() {
				err = p.err("system does not support ctime")
			} else {
				err = p.olderThanNotOlderThanCheck(p.CtimeOlderThan, p.CtimeNotOlderThan, fi.ChangeTime())
			}
		}
		if err == nil && (p.BtimeOlderThan != 0 || p.BtimeNotOlderThan != 0) {
			if !fi.HasBirthTime() {
				err = p.err("system does not support btime")
			} else {
				err = p.olderThanNotOlderThanCheck(p.BtimeOlderThan, p.BtimeNotOlderThan, fi.BirthTime())
			}
		}
	}

	return err
}

func (p *FileProbe) RunLoop(stop <-chan struct{}, res ProbeResultReceiver) error {
	// validate
	if p.Path == "" && p.PathNotExists == "" && p.PathIsDirectory == "" && p.PathIsFile == "" {
		return p.err("one of path, path_exists, path_is_file or path_is_directory parameters is required")
	}

	for {
		var err error

		if p.PathNotExists != "" {
			_, err = os.Stat(p.PathNotExists)
			if err == nil {
				err = p.err("path exists")
			} else if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		}

		if err == nil && p.PathIsFile != "" {
			var fi os.FileInfo
			fi, err = os.Stat(p.PathIsFile)
			if fi.IsDir() {
				err = p.err("path is not a file")
			}
		}

		if err == nil && p.PathIsDirectory != "" {
			var fi os.FileInfo
			fi, err = os.Stat(p.PathIsDirectory)
			if !fi.IsDir() {
				err = p.err("path is not a directory")
			}
		}

		if err == nil {
			err = p.runFileExistenceAndTimestampChecks()
		}

		go res.ProbeResult(p, p.runDuration(), err)

		if !p.nextRun(stop) {
			break
		}
	}

	return nil
}
