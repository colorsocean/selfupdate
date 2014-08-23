package selfupdate

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"bitbucket.org/kardianos/osext"

	. "github.com/colorsocean/selfupdate"
	. "github.com/colorsocean/selfupdate/common"
)

const ()

var ()

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

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

	err = decodeFrom(&ut.info, ut.infoFilePath)
	panicOn(err)

	ut.issuerExeBackup = ut.info.IssuerExe + ".bak"

	ut.UpdateDir = filepath.Join(ut.selfDir, "update")
	ut.IssuerExe = ut.info.IssuerExe

	return ut
}

func unzip(archivePath, targetDir string) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(targetDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
		} else {
			reader, err := file.Open()
			if err != nil {
				return err
			}
			defer reader.Close()

			fileDir, _ := filepath.Split(filePath)
			err = os.MkdirAll(fileDir, file.Mode())
			if err != nil {
				return err
			}

			writer, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return err
			}
			defer writer.Close()

			_, err = io.Copy(writer, reader)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
