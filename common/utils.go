package common

import (
	"archive/zip"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	FileTsFormat = "2006-01-02T15-04-05-999999999Z0700"
)

func ExtractOneToBase(file *zip.File, basePath string) error {
	path := filepath.Join(basePath, file.Name)
	return ExtractOne(file, path)
}

func ExtractOne(file *zip.File, path string) error {
	if file.FileInfo().IsDir() {
		os.MkdirAll(path, file.Mode())
	} else {
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()

		fileDir, _ := filepath.Split(path)
		err = os.MkdirAll(fileDir, file.Mode())
		if err != nil {
			return err
		}

		writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
	}

	return nil
}

func CopyFile(source string, dest string) (err error) {
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

func TrimExeExt(p string) string {
	if runtime.GOOS == "windows" {
		return strings.TrimSuffix(p, ".exe")
	} else {
		return p
	}
}

func EnsureExeExt(p string) string {
	if runtime.GOOS == "windows" {
		if strings.HasSuffix(p, ".exe") {
			return p
		} else {
			return p + ".exe"
		}
	} else {
		return p
	}
}

func GetJson(v interface{}) string {
	json, _ := json.MarshalIndent(v, "", "    ")
	return string(json)
}

func PanicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func EncodeTo(v interface{}, fileName string) error {
	data, err := Encode(v)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, 0)
	if err != nil {
		return err
	}
	return nil
}

func DecodeFrom(v interface{}, fileName string) error {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	err = Decode(v, data)
	if err != nil {
		return err
	}
	return nil
}

func Encode(v interface{}) ([]byte, error) {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(v)
	return buffer.Bytes(), err
}

func Decode(v interface{}, data []byte) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(v)
	return err
}
