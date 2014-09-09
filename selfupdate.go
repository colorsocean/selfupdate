//
//
package selfupdate

// todo: [ ] Reduce amount of stored data, replace with calculations
// todo: [ ] Prevent simulateous update attempts
// todo: [ ] Make Version accept letters

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"bitbucket.org/kardianos/osext"

	. "github.com/colorsocean/selfupdate/common"
)

var (
	env SelfupdateEnv

	ErrNoUptoolExe = errors.New("Missing uptool executable in update archive")
)

//> Move to internal
type SelfupdateEnv struct {
	IssuerVersion     Version
	MaxUpdateAttempts int

	log *log.Logger
}

func (this SelfupdateEnv) IssuerExe() string {
	exe, err := osext.Executable()
	PanicOn(err)
	return exe
}

func (this SelfupdateEnv) IssuerDir() string {
	dir, _ := filepath.Split(this.IssuerExe())
	return dir
}

func (this SelfupdateEnv) IssuerName() string {
	_, name := filepath.Split(this.IssuerExe())
	return TrimExeExt(name)
}

func (this SelfupdateEnv) SelfupdateInfoPath() string {
	return TrimExeExt(this.IssuerExe()) + SelfupdateInfoExt
}

func (this SelfupdateEnv) TargetDir() string {
	return this.IssuerDir()
}

func (this SelfupdateEnv) UpdatesCacheDir() string {
	return filepath.Join(this.TargetDir(), UpdatesCacheDirName)
}

func (this SelfupdateEnv) SelfupdateLogPath() string {
	return filepath.Join(this.UpdatesCacheDir(), this.IssuerName()+".log")
}

func (this SelfupdateEnv) UpdateDir(ver Version) string {
	return filepath.Join(this.UpdatesCacheDir(), fmt.Sprintf("%s-%s", this.IssuerName(), ver))
}

/****************************************************************
** Types
********/

type Options struct {
	Version           Version
	MaxUpdateAttempts int
}

/****************************************************************
** Logic
********/

func Init(options Options) func() {
	env = SelfupdateEnv{}

	err := os.MkdirAll(env.UpdatesCacheDir(), 0)
	PanicOn(err)

	logFile, err := os.Create(env.SelfupdateLogPath())
	PanicOn(err)

	env.log = log.New(logFile, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	env.IssuerVersion = options.Version
	env.MaxUpdateAttempts = options.MaxUpdateAttempts
	if env.MaxUpdateAttempts <= 0 {
		env.MaxUpdateAttempts = 3
	}

	cleanup := func() {
		if logFile != nil {
			logFile.Close()
		}
	}

	if checkUpdateReady() {
		cleanup()
		os.Exit(0)
	}

	return func() {
		defer cleanup()
		checkUpdateReady()
	}
}

func checkUpdateReady() (mustExit bool) {
	env.log.Println("Checking necessity of update")
	si := &SelfupdateInfo{}
	err := DecodeFrom(si, env.SelfupdateInfoPath())
	if os.IsNotExist(err) {
		env.log.Println("Update is not necessary")
		return false
	}
	PanicOn(err)

	isGreater, err := Version(si.UpdateVersion).IsGreater(env.IssuerVersion)
	PanicOn(err)
	if !isGreater {
		env.log.Println("Update version is not recent")
		cleanupSelfupdate(si)
		return false
	}

	if si.UpdateAttempt >= env.MaxUpdateAttempts {
		env.log.Println("Reached maximum number of update attempts")
		cleanupSelfupdate(si)
		return false
	}

	si.UpdateAttempt++
	err = EncodeTo(si, env.SelfupdateInfoPath())
	PanicOn(err)

	env.log.Println("Sterting uptool")
	err = exec.Command(si.UptoolPath()).Start()
	PanicOn(err)

	return true
}

func prepareSelfupdate(version Version, autoStart bool) (si *SelfupdateInfo, f *os.File, err error) {
	env.log.Println("Preparing selfupdate")
	si = &SelfupdateInfo{}
	si.UpdateDir = env.UpdateDir(version)
	si.UpdateVersion = string(version)
	si.IssuerVersion = string(env.IssuerVersion)
	si.StartAfterUpdate = autoStart

	env.log.Println("Creating update directory")
	err = os.MkdirAll(si.UpdateDir, 0)
	if err != nil {
		env.log.Println("Error creating update directory:", err.Error())
		return nil, nil, err
	}

	env.log.Println("Creating update archive file")
	f, err = os.Create(si.UpdateArchivePath())
	if err != nil {
		env.log.Println("Error creating update archive file:", err.Error())
		return nil, nil, err
	}

	return si, f, nil
}

func continueSelfupdate(si *SelfupdateInfo) error {
	env.log.Println("Extracting uptool")
	err := extractUptool(si)
	if err != nil {
		env.log.Println("Error extracting uptool:", err.Error())
		return err
	}

	env.log.Println("Updating selfupdate info")
	err = EncodeTo(si, env.SelfupdateInfoPath())
	if err != nil {
		env.log.Println("Error updating selfupdate info:", err.Error())
		return err
	}

	uti := &UptoolInfo{}
	uti.TargetDir = env.TargetDir()
	uti.IssuerExe = env.IssuerExe()
	uti.StartAfterUpdate = si.StartAfterUpdate

	env.log.Println("Writing uptool info")
	err = EncodeTo(uti, si.UptoolInfoPath())
	if err != nil {
		env.log.Println("Error writing uptool info:", err.Error())
		return err
	}

	return nil
}

func cleanupSelfupdate(si *SelfupdateInfo) error {
	return nil

	err := os.Remove(env.SelfupdateInfoPath())
	if err != nil {
		env.log.Println("Cleanup error:", err.Error())
		return err
	}

	env.log.Println("Selfupdate cleanup")
	err = os.RemoveAll(si.UpdateDir)
	if err != nil {
		env.log.Println("Cleanup error:", err.Error())
		return err
	}

	env.log.Println("Cleanup successful")
	return nil
}

/****************************************************************
** Update interface
********/

type updateWriteCloser struct {
	si *SelfupdateInfo
	f  *os.File
}

func (this *updateWriteCloser) Write(p []byte) (n int, err error) {
	n, err = this.f.Write(p)
	if err != nil {
		env.log.Println("Error writing update file:", err.Error())
		cleanupSelfupdate(this.si)
		return
	}
	return
}

func (this *updateWriteCloser) Close() (err error) {
	env.log.Println("Closing update file")
	err = this.f.Close()
	if err != nil {
		env.log.Println("Error closing update file:", err.Error())
		cleanupSelfupdate(this.si)
		return
	}

	env.log.Println("Continuing selfupdate")
	err = continueSelfupdate(this.si)
	if err != nil {
		env.log.Println("Continuing selfupdate error", err)
		cleanupSelfupdate(this.si)
		return
	}

	return
}

func ViaWriteCloser(version Version, autoStart bool) (io.WriteCloser, error) {
	env.log.Println("Preparing selfupdate")
	si, f, err := prepareSelfupdate(version, autoStart)
	if err != nil {
		env.log.Println("Preparing selfupdate error", err)
		cleanupSelfupdate(si)
		return nil, err
	}
	w := &updateWriteCloser{si, f}
	return w, nil
}

/****************************************************************
** Utils
********/

func extractUptool(si *SelfupdateInfo) error {
	zipReader, err := zip.OpenReader(si.UpdateArchivePath())
	if err != nil {
		return err
	}
	defer zipReader.Close()

	uptoolName := EnsureExeExt(UptoolExeName)
	env.log.Println("uptoolName:", uptoolName)
	for _, file := range zipReader.File {
		env.log.Println("zipfile:", file.Name)
		if file.Name == uptoolName {
			return ExtractOne(file, si.UptoolPath())
		}
	}

	return ErrNoUptoolExe
}
