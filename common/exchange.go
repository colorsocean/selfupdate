package common

import "path/filepath"

const (
	SelfupdateInfoExt = ".selfupdate"

	UpdatesCacheDirName = "updates"
	UpdateArchiveName   = "update"
	UptoolExeName       = "uptool"
	UptoolInfoName      = "selfupdate"
)

/****************************************************************
** UptoolInfo
********/

type UptoolInfo struct {
	TargetDir      string
	IssuerExe      string
	RunAfterUpdate bool
}

func (this UptoolInfo) SelfupdateInfoPath() string {
	return TrimExeExt(this.IssuerExe) + SelfupdateInfoExt
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
