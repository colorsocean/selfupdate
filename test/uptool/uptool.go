package main

import (
	"path/filepath"
	"time"

	"github.com/colorsocean/selfupdate"
)

func main() {
	tool := selfupdate.UpTool()
	tool.Log.Debugln("Uptool started")

	if !tool.WaitRemoveTarget(2 * time.Second) {
		tool.Log.Debugln("Issuer not removed")
		return
	}
	tool.Log.Debugln("Issuer removed")

	err := tool.CopyFile(filepath.Join(tool.UpdateDir, "test.exe"), tool.IssuerExe)
	if err != nil {
		tool.Log.Debugln("Issuer not replaced", err.Error())
		return
	}
	tool.Log.Debugln("Issuer replaced")

	tool.Success()
}
