package selfupdate

import (
	"archive/zip"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"bitbucket.org/kardianos/osext"

	. "github.com/colorsocean/selfupdate/common"
)

const ()

var (
	env = uptoolEnv{}
)

type uptoolEnv struct {
	UptoolInfo

	IssuerExeIsAService bool
	IssuerExeDeleted    bool
	Done                bool

	logFile *os.File
}

func (this uptoolEnv) SelfExe() string {
	exe, err := osext.Executable()
	PanicOn(err)
	return exe
}

func (this uptoolEnv) SelfDir() string {
	dir, _ := filepath.Split(this.SelfExe())
	return dir
}

func (this uptoolEnv) UptoolInfoPath() string {
	return filepath.Join(this.SelfDir(), UptoolInfoName)
}

func (this uptoolEnv) UptoolLogPath() string {
	return filepath.Join(this.SelfDir(), UptoolLogName)
}

func (this uptoolEnv) UpdateArchivePath() string {
	return filepath.Join(this.SelfDir(), UpdateArchiveName)
}

func init() {
	var err error
	env.logFile, err = os.Create(env.UptoolLogPath())
	PanicOn(err)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(env.logFile)
	defer func() {
		if rec := recover(); rec != nil {
			log.Println("Error initializing uptool", rec)
		}
	}()
	err = DecodeFrom(&env.UptoolInfo, env.UptoolInfoPath())
	PanicOn(err)
	log.Println("Uptool started")
}

func done() {
	env.Done = true
	if env.IssuerExeDeleted {
		err := os.Remove(env.IssuerExeBak())
		if err != nil {
			log.Println("Warning:", err.Error())
		}
	}

	err := os.Remove(env.SelfupdateInfoPath())
	PanicOn(err)

	finally()
	os.Exit(0)
}

func failure() {
	if env.IssuerExeDeleted {
		err := os.Rename(env.IssuerExeBak(), env.IssuerExe)
		PanicOn(err)
	}

	finally()
	os.Exit(1)
}

func finally() {
	if env.StartAfterUpdate {
		if env.IssuerExeIsAService {
			err := exec.Command(env.IssuerExe, "start").Run()
			PanicOn(err)
		} else {
			err := exec.Command(env.IssuerExe).Start()
			PanicOn(err)
		}
	}
}

func Recover() {
	if rec := recover(); rec != nil {
		log.Println("Uptool failed, preparing simple rollback. Message:", rec)
		if env.Done {
			log.Println("Failure after Done()! So, sometimes shit happens...")
		} else {
			failure()
		}
		return
	}

	done()
}

func Done() {
	done()
}

func Failure(msg ...string) {
	m := "Uptool failed"
	if len(msg) > 0 {
		m = msg[0]
	}
	panic(m)
}

func IssuerExeIsAService(really bool) {
	env.IssuerExeIsAService = really
}

func StopIssuerExeServiceWithin(d time.Duration) {
	//err := exec.Command(env.IssuerExe, "stop").Run()
	//PanicOn(err)
}

func RemoveIssuerExeWithin(d time.Duration) {
	log.Println("Removing issuer exe")
	start := time.Now()
	for {
		time.Sleep(100 * time.Millisecond)

		_, err := os.Stat(env.IssuerExe)
		PanicOn(err)
		err = os.Remove(env.IssuerExeBak())
		err = os.Rename(env.IssuerExe, env.IssuerExeBak())
		if err == nil {
			log.Println("Issuer exe deleted successfuly")
			env.IssuerExeDeleted = true
			return
		}

		now := time.Now()
		if now.Sub(start) >= d {
			panic("Issuer exe is not deleted within timeout period")
		}
	}
}

func ReplaceIssuerExeWith(name string) {
	log.Println("Replacing issuer exe with", name)
	zipReader, err := zip.OpenReader(env.UpdateArchivePath())
	PanicOn(err)
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if file.Name == name {
			err = ExtractOne(file, env.IssuerExe)
			PanicOn(err)
			log.Println("Issuer exe replaced successfuly")
			break
		}
	}
}
