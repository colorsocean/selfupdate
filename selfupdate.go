package selfupdate

import (
	"archive/zip"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"bitbucket.org/kardianos/osext"
	"github.com/Sirupsen/logrus"
)

var (
	app appData
	log *logrus.Logger
)

const (
	selfupdateInfoFileExt = ".selfupdate"
	uptoolInfoFileName    = "selfupdate"
	uptoolImageName       = "uptool"
	updateArchiveFileName = "update.zip"
	selfupdateDirName     = "selfupdate"
	updateExtractDirName  = "update"

	fileTsFormat = "2006-01-02T15-04-05-999999999Z0700"
)

/****************************************************************
** Types
********/

type Options struct {
	Version           Version
	MaxUpdateAttempts int
}

type appData struct {
	Ops Options

	SelfDir       string
	SelfPath      string
	SelfNameNoExt string
	SelfName      string

	SelfupdateInfoFilePath string

	UptoolImageFileName string
}

// todo: Reduce amount of stored data, replace with calculations
type selfupdateInfo struct {
	UpdateDir          string
	UpdateExtractDir   string
	UpdateArchivePath  string
	UptoolPath         string
	UptoolInfoFilePath string
	UptoolExtractPath  string
	IssuerVersion      Version
	UpdateVersion      Version
	UpdateAttempt      int

	RunAfterUpdate bool

	updateArchiveFile *os.File
}

type uptoolInfo struct {
	IssuerDir              string
	IssuerExe              string
	IssuerVersion          Version
	UpdateVersion          Version
	RunAfterUpdate         bool
	SelfupdateInfoFilePath string
}

/****************************************************************
** Logic
********/

func Init(options Options) func() {
	app = appData{
		Ops: options,
	}

	log = logrus.New()
	log.Out = os.Stderr
	log.Level = logrus.DebugLevel
	log.Hooks.Add(&SimpleLogrusHook{
		func(entry *logrus.Entry) {
			entry.Data["emmiter"] = "module"
			entry.Data["name"] = "selfupdate"
			entry.Data["pid"] = os.Getpid()
		},
	})

	var err error

	app.SelfDir, err = osext.ExecutableFolder()
	panicOn(err)
	app.SelfPath, err = osext.Executable()
	panicOn(err)

	app.SelfName = strings.TrimPrefix(app.SelfPath, app.SelfDir)
	app.SelfNameNoExt = strings.TrimSuffix(app.SelfName, ".exe")

	app.SelfupdateInfoFilePath = filepath.Join(app.SelfDir, fmt.Sprintf("%s%s", app.SelfNameNoExt, selfupdateInfoFileExt))

	app.UptoolImageFileName = uptoolImageName
	if runtime.GOOS == "windows" {
		app.UptoolImageFileName += ".exe"
	}

	//exeExtIfAny := ""
	//if strings.HasSuffix(selfPath, ".exe") {
	//	exeExtIfAny = ".exe"
	//}
	//selfupdateFileName = filepath.Join(selfDir, fmt.Sprintf("%s%s", selfNameNoExt, exeExtIfAny))

	log.Debugln("options:", getJson(app))

	if checkUpdateReady() {
		os.Exit(0)
	}

	return func() {
		checkUpdateReady()
	}
}

func checkUpdateReady() (mustExit bool) {
	si := &selfupdateInfo{}
	err := decodeFrom(si, app.SelfupdateInfoFilePath)
	if os.IsNotExist(err) {
		return false
	}

	if si.UpdateAttempt >= app.Ops.MaxUpdateAttempts {
		os.Remove(app.SelfupdateInfoFilePath)
		cleanupSelfupdate(si)
		return false
	}

	si.UpdateAttempt++
	err = encodeTo(si, app.SelfupdateInfoFilePath)
	panicOn(err)

	err = exec.Command(si.UptoolPath).Start()
	panicOn(err)

	return true
}

func prepareSelfupdate(version Version, autoStart bool) (si *selfupdateInfo, err error) {
	si = &selfupdateInfo{}
	si.UpdateDir = generateUpdateDir()
	si.UpdateExtractDir = filepath.Join(si.UpdateDir, updateExtractDirName)
	si.UpdateArchivePath = filepath.Join(si.UpdateDir, updateArchiveFileName)
	si.UptoolExtractPath = filepath.Join(si.UpdateExtractDir, app.UptoolImageFileName)
	si.UptoolPath = filepath.Join(si.UpdateDir, app.UptoolImageFileName)
	si.UptoolInfoFilePath = filepath.Join(si.UpdateDir, uptoolInfoFileName)

	si.IssuerVersion = app.Ops.Version
	si.UpdateVersion = version
	si.RunAfterUpdate = autoStart

	err = os.MkdirAll(si.UpdateExtractDir, 0)
	if err != nil {
		return nil, err
	}

	si.updateArchiveFile, err = os.Create(si.UpdateArchivePath)
	if err != nil {
		return nil, err
	}

	return si, nil
}

func continueSelfupdate(si *selfupdateInfo) error {
	err := unzip(si.UpdateArchivePath, si.UpdateExtractDir)
	if err != nil {
		return err
	}
	err = os.Rename(si.UptoolExtractPath, si.UptoolPath)
	if err != nil {
		return err
	}

	err = encodeTo(si, app.SelfupdateInfoFilePath)
	if err != nil {
		return err
	}

	uti := &uptoolInfo{}
	uti.IssuerDir = app.SelfDir
	uti.IssuerExe = app.SelfPath
	uti.IssuerVersion = si.IssuerVersion
	uti.UpdateVersion = si.UpdateVersion
	uti.SelfupdateInfoFilePath = app.SelfupdateInfoFilePath
	uti.RunAfterUpdate = si.RunAfterUpdate

	err = encodeTo(uti, si.UptoolInfoFilePath)
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
	si *selfupdateInfo
}

func (this *updateWriteCloser) Write(p []byte) (n int, err error) {
	n, err = this.si.updateArchiveFile.Write(p)
	if err != nil {
		cleanupSelfupdate(this.si)
		return
	}
	return
}

func (this *updateWriteCloser) Close() (err error) {
	err = this.si.updateArchiveFile.Close()
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
	si, err := prepareSelfupdate(version, autoStart)
	if err != nil {
		cleanupSelfupdate(si)
		return nil, err
	}
	w := &updateWriteCloser{si}
	return w, nil
}

/****************************************************************
** Utils
********/

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func getJson(v interface{}) string {
	result, _ := json.MarshalIndent(v, "", "  ")
	return string(result)
}

func generateUpdateDir() string {
	return filepath.Join(app.SelfDir, selfupdateDirName,
		fmt.Sprintf("%s-%s", app.SelfNameNoExt, time.Now().UTC().Format(fileTsFormat)))
}

func getSelfupdateInfoFilePath() string {
	return filepath.Join(app.SelfDir, fmt.Sprintf("%s%s", app.SelfNameNoExt, selfupdateInfoFileExt))
}

func encodeTo(v interface{}, fileName string) error {
	data := encode(v)
	err := ioutil.WriteFile(fileName, data, 0)
	if err != nil {
		return err
	}
	return nil
}

func decodeFrom(v interface{}, fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	decode(v, data)
	return nil
}

func encode(v interface{}) []byte {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(v)
	panicOn(err)
	return buffer.Bytes()
}

func decode(v interface{}, data []byte) {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(v)
	panicOn(err)
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

type SimpleLogrusHook struct {
	OnFire func(*logrus.Entry)
}

func (this *SimpleLogrusHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func (this *SimpleLogrusHook) Fire(entry *logrus.Entry) error {
	this.OnFire(entry)
	return nil
}
