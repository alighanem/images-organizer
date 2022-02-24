package main

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
)

var ImagesFolderPath string = ""
var layoutISO = "2006-01-02"

func main() {

	ImagesFolderPath = os.Getenv("IMAGES_FOLDER")
	if len(ImagesFolderPath) == 0 {
		log.Fatal("Images folder is empty")
	}

	var files []fs.FileInfo

	err := filepath.WalkDir(ImagesFolderPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		files = append(files, info)
		return nil
	})
	if err != nil {
		log.Fatalln("error reading file path", "err", err)
	}

	if len(files) == 0 {
		log.Println("no files found in the folder")
		return
	}

	for _, file := range files {
		modifiedDate := file.ModTime().UTC()
		firstDay := time.Date(modifiedDate.Year(), modifiedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		folderName := firstDay.UTC().Format(layoutISO)
		folderPath := path.Join(ImagesFolderPath, folderName)
		_, err := os.Stat(folderPath)
		if os.IsNotExist(err) {
			_ = os.Mkdir(folderPath, 0755)
		}
		oldPath := path.Join(ImagesFolderPath, file.Name())
		newPath := path.Join(folderPath, file.Name())

		err = os.Rename(oldPath, newPath)
		if err != nil {
			log.Fatal("cannot move file", "name", file.Name(), "old path", oldPath, "new path", newPath)
		}
	}

	log.Println("finished")

}
