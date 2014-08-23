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
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"bitbucket.org/kardianos/osext"

	"github.com/Sirupsen/logrus"
	. "github.com/colorsocean/selfupdate/common"
)

var (
	env SelfupdateEnv

	ErrNoUptoolExe = errors.New("Missing uptool executable in update archive")
)

type SelfupdateEnv struct {
	IssuerVersion     Version
	MaxUpdateAttempts int

	log *logrus.Logger
}

func (this SelfupdateEnv) IssuerExe() string {
	exe, err = osext.Executable()
	PanicOn(err)
	return exe
}

func (this SelfupdateEnv) IssuerDir() string {
	dir, _ = filepath.Split(this.IssuerExe())
	return dir
}

func (this SelfupdateEnv) IssuerName() string {
	_, name = filepath.Split(this.IssuerExe())
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

	env.log = logrus.New()
	env.log.Out = os.Stderr
	env.log.Level = logrus.DebugLevel
	env.log.Hooks.Add(&simpleLogrusHook{
		func(entry *logrus.Entry) {
			entry.Data["emmiter"] = "module"
			entry.Data["name"] = "selfupdate"
			entry.Data["pid"] = os.Getpid()
		},
	})

	var err error

	env.IssuerVersion = options.Version
	env.MaxUpdateAttempts = options.MaxUpdateAttempts
	if env.MaxUpdateAttempts <= 0 {
		env.MaxUpdateAttempts = 3
	}

	if checkUpdateReady() {
		os.Exit(0)
	}

	return func() {
		checkUpdateReady()
	}
}

func checkUpdateReady() (mustExit bool) {
	si := &selfupdateInfo{}
	err := DecodeFrom(si, env.SelfupdateInfoFilePath)
	if os.IsNotExist(err) {
		return false
	}

	if si.UpdateAttempt >= env.Ops.MaxUpdateAttempts {
		os.Remove(env.SelfupdateInfoFilePath)
		cleanupSelfupdate(si)
		return false
	}

	si.UpdateAttempt++
	err = EncodeTo(si, env.SelfupdateInfoFilePath)
	PanicOn(err)

	err = exec.Command(si.UptoolPath).Start()
	PanicOn(err)

	return true
}

func prepareSelfupdate(version Version, autoStart bool) (si *selfupdateInfo, f *os.File, err error) {
	si = &SelfupdateInfo{}
	si.UpdateDir = env.UpdateDir(version)
	si.UpdateVersion = version
	si.IssuerVersion = env.IssuerVersion
	si.StartAfterUpdate = autoStart

	err = os.MkdirAll(si.UpdateDir, 0)
	if err != nil {
		return nil, nil, err
	}

	f, err = os.Create(si.UpdateArchivePath())
	if err != nil {
		return nil, nil, err
	}

	return si, f, nil
}

func continueSelfupdate(si *selfupdateInfo) error {
	err := extractUptool(si)
	if err != nil {
		return err
	}

	err = EncodeTo(si, env.SelfupdateInfoPath())
	if err != nil {
		return err
	}

	uti := &UptoolInfo{}
	uti.IssuerDir = env.SelfDir
	uti.IssuerExe = env.SelfPath
	uti.IssuerVersion = si.IssuerVersion
	uti.UpdateVersion = si.UpdateVersion
	uti.SelfupdateInfoFilePath = env.SelfupdateInfoFilePath
	uti.RunAfterUpdate = si.RunAfterUpdate

	err = EncodeTo(uti, si.UptoolInfoPath())
	if err != nil {
		return err
	}

	return nil
}

func cleanupSelfupdate(si *selfupdateInfo) error {
	err := os.RemoveAll(si.UpdateDir)
	if err != nil {
		return err
	}
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
	n, err = this.f.updateArchiveFile.Write(p)
	if err != nil {
		cleanupSelfupdate(this.si)
		return
	}
	return
}

func (this *updateWriteCloser) Close() (err error) {
	err = this.f.updateArchiveFile.Close()
	if err != nil {
		cleanupSelfupdate(this.si)
		return
	}

	err = continueSelfupdate(this.si)
	if err != nil {
		cleanupSelfupdate(this.si)
		return
	}

	return
}

func ViaWriteCloser(version Version, autoStart bool) (io.WriteCloser, error) {
	si, f, err := prepareSelfupdate(version, autoStart)
	if err != nil {
		cleanupSelfupdate(si)
		return nil, err
	}
	w := &updateWriteCloser{si, f}
	return w, nil
}

/****************************************************************
** Utils
********/

func generateUpdateDir() string {
	return filepath.Join(env.SelfDir, selfupdateDirName,
		fmt.Sprintf("%s-%s", env.SelfNameNoExt, time.Now().UTC().Format(fileTsFormat)))
}

func getSelfupdateInfoFilePath() string {
	return filepath.Join(env.SelfDir, fmt.Sprintf("%s%s", env.SelfNameNoExt, selfupdateInfoFileExt))
}

func extractUptool(si *SelfupdateInfo) error {
	zipReader, err := zip.OpenReader(si.UpdateArchivePath())
	if err != nil {
		return err
	}
	defer zipReader.Close()

	uptoolName := EnsureExeExt(UptoolExeName)
	for _, file := range zipReader.File {
		if file.Name == uptoolName {
			return ExtractOne(file, si.UptoolPath())
		}
	}

	return ErrNoUptoolExe
}

type simpleLogrusHook struct {
	OnFire func(*logrus.Entry)
}

func (this *simpleLogrusHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func (this *simpleLogrusHook) Fire(entry *logrus.Entry) error {
	this.OnFire(entry)
	return nil
}
