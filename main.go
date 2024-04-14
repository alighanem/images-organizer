package main

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

var picturesFolderPath string = ""
var rootImagesFolderPath string = ""
var layoutISO = "2006-01"

func main() {

	picturesFolderPath = os.Getenv("PICTURES_FOLDER")
	if picturesFolderPath == "" {
		log.Fatal("Images folder is empty")
	}

	rootImagesFolderPath = filepath.Dir(picturesFolderPath)

	var files []fs.FileInfo

	err := filepath.WalkDir(picturesFolderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Println("cannot read", "err", err)
			return err
		}

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

		yearFolder := path.Join(rootImagesFolderPath, strconv.Itoa(modifiedDate.Year()))
		_, err := os.Stat(yearFolder)
		if os.IsNotExist(err) {
			err = os.Mkdir(yearFolder, 0755)
			if err != nil {
				log.Fatalln("creating year folder", "folder_path", yearFolder)
			}
		}

		folderName := modifiedDate.Format(layoutISO)
		folderPath := path.Join(yearFolder, folderName)
		_, err = os.Stat(folderPath)
		if os.IsNotExist(err) {
			err = os.Mkdir(folderPath, 0755)
			if err != nil {
				log.Fatalln("creating image folder", "folder_path", yearFolder)
			}
		}

		oldPath := path.Join(picturesFolderPath, file.Name())
		newPath := path.Join(folderPath, file.Name())

		err = os.Rename(oldPath, newPath)
		if err != nil {
			log.Println("cannot move file", "name", file.Name(), "old path", oldPath, "new path", newPath, "err", err)
			continue
		}
	}

	log.Println("finished")

}
