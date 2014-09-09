package common

import "path/filepath"

const (
	SelfupdateInfoExt = ".selfupdate"

	UpdatesCacheDirName = "selfupdate"
	UpdateArchiveName   = "update"
	UptoolExeName       = "uptool"
	UptoolInfoName      = "selfupdate"
	UptoolLogName       = "uptool.log"
)

/****************************************************************
** UptoolInfo
********/

type UptoolInfo struct {
	TargetDir        string
	IssuerExe        string
	StartAfterUpdate bool
}

func (this UptoolInfo) IssuerExeBak() string {
	return this.IssuerExe + ".bak"
}

func (this UptoolInfo) SelfupdateInfoPath() string {
	return TrimExeExt(this.IssuerExe) + SelfupdateInfoExt
}

func (this UptoolInfo) IssuerDir() string {
	dir, _ := filepath.Split(this.IssuerExe)
	return dir
}

/****************************************************************
** SelfupdateInfo
********/

type SelfupdateInfo struct {
	UpdateDir        string
	UpdateVersion    string // selfupdate.Version
	IssuerVersion    string // selfupdate.Version
	StartAfterUpdate bool
	UpdateAttempt    int
}

func (this SelfupdateInfo) UpdateArchivePath() string {
	return filepath.Join(this.UpdateDir, UpdateArchiveName)
}

func (this SelfupdateInfo) UptoolPath() string {
	return filepath.Join(this.UpdateDir, EnsureExeExt(UptoolExeName))
}

func (this SelfupdateInfo) UptoolInfoPath() string {
	return filepath.Join(this.UpdateDir, UptoolInfoName)
}
