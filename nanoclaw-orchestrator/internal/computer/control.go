package computer

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-vgo/robotgo"
)

func TakeScreenshot(savePath string) error {
	img, err := robotgo.CaptureImg()
	if err != nil {
		return err
	}
	return robotgo.Save(img, savePath)
}

func TypeText(text string) {
	robotgo.TypeStr(text)
}

func ClickAt(x, y int) {
	robotgo.Move(x, y)
	robotgo.Click("left", false)
}

func DoubleClickAt(x, y int) {
	robotgo.Move(x, y)
	robotgo.Click("left", true)
}

func KeyboardPress(key string) {
	robotgo.KeyTap(key)
}

func MouseScroll(x, y int, direction string) {
	robotgo.Move(x, y)
	if direction == "up" {
		robotgo.ScrollDir(10, "up")
	} else {
		robotgo.ScrollDir(10, "down")
	}
}

func OpenFolder(path string) error {
	return exec.Command("open", path).Run()
}

func ExtractZip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		outFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func WaitForFile(path string, timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutChan:
			return false
		case <-ticker.C:
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
	}
}
