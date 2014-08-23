package selfupdate

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"bitbucket.org/kardianos/osext"
	"github.com/Sirupsen/logrus"
)

type uptool struct {
	UpdateDir string
	IssuerExe string

	selfDir      string
	selfExe      string
	infoFilePath string
	logFilePath  string
	info         uptoolInfo

	issuerExeBackup string

	logFile *os.File
	Log     *logrus.Logger
}

func (this *uptool) WaitRemoveTarget(d time.Duration) (deleted bool) {
	start := time.Now()
	for {
		time.Sleep(500 * time.Millisecond)

		_, err := os.Stat(this.IssuerExe)
		panicOn(err)
		os.Remove(this.issuerExeBackup)
		err = os.Rename(this.IssuerExe, this.issuerExeBackup)
		if err == nil {
			return true
		}

		now := time.Now()
		if now.Sub(start) >= d {
			return false
		}
	}
}

func (this *uptool) Success() {
	if this.info.RunAfterUpdate {
		err := exec.Command(this.IssuerExe).Start()
		panicOn(err)
	}
	os.Remove(this.info.SelfupdateInfoFilePath)
	os.Exit(0)
}

func (this *uptool) CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return
}

func UpTool() *uptool {
	ut := &uptool{}

	var err error

	ut.selfDir, err = osext.ExecutableFolder()
	panicOn(err)
	ut.selfExe, err = osext.Executable()
	panicOn(err)

	ut.infoFilePath = filepath.Join(ut.selfDir, uptoolInfoFileName)
	ut.logFilePath = filepath.Join(ut.selfDir, "uptool.log")

	ut.logFile, err = os.Create(ut.logFilePath)
	panicOn(err)

	ut.Log = logrus.New()
	ut.Log.Out = ut.logFile
	ut.Log.Level = logrus.DebugLevel
	ut.Log.Hooks.Add(&SimpleLogrusHook{
		func(entry *logrus.Entry) {
			entry.Data["emmiter"] = "application"
			entry.Data["name"] = "uptool"
			entry.Data["pid"] = os.Getpid()
		},
	})

	err = decodeFrom(&ut.info, ut.infoFilePath)
	panicOn(err)

	ut.issuerExeBackup = ut.info.IssuerExe + ".bak"

	ut.UpdateDir = filepath.Join(ut.selfDir, "update")
	ut.IssuerExe = ut.info.IssuerExe

	return ut
}
