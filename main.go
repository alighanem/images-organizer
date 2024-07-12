package main

import (
	"io/fs"
	"log"
	"os"
	"path"
	"strconv"
	"time"
)

func main() {

	// This folder is the folder where are located the images we want to move
	sourceFolderPath := os.Getenv("PICTURES_FOLDER")
	if sourceFolderPath == "" {
		log.Fatal("Source Images folder is not defrined")
	}

	// use path join to format correctly the source folder path
	sourceFolderPath = path.Join(sourceFolderPath)

	// Read the folder to get all files in this folder
	entries, err := os.ReadDir(sourceFolderPath)
	if err != nil {
		log.Fatalln("failed reading folder", "folder_path", sourceFolderPath, "err", err)
		return
	}

	var files []fs.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			log.Fatalln("failed reading file", "err", err, "entry", entry.Name())
		}

		files = append(files, info)
	}

	if len(files) == 0 {
		log.Println("no files found in the folder")
		return
	}

	// Get the parent folder of the folder source
	// For example if the folder source is C:/images/pellicule
	// the parent folder will be C:/images
	rootPath := path.Dir(sourceFolderPath)
	i := 0

	for _, file := range files {

		// Compute the destination folder of the image based on its modification date
		// The destination folder will be like C:/images/2024/2024-06-15
		destinationFolder := computeDestinationFolder(rootPath, file.ModTime())
		_, err := os.Stat(destinationFolder)
		if os.IsNotExist(err) {
			// Create the destination folder if it does not existy
			err = os.Mkdir(destinationFolder, 0755)
			if err != nil {
				log.Fatalln("creating image folder", "folder_path", destinationFolder)
			}
		}

		oldPath := path.Join(sourceFolderPath, file.Name())
		newPath := path.Join(destinationFolder, file.Name())

		err = os.Rename(oldPath, newPath)
		if err != nil {
			log.Println("cannot move file", "name", file.Name(), "old path", oldPath, "new path", newPath, "err", err)
			continue
		}

		// Sleep to pause the system
		if i%5 == 0 {
			time.Sleep(100 * time.Millisecond)
		}

		i++
	}

	log.Println("finished")
}

func computeDestinationFolder(imagesFolderPath string, fileDate time.Time) string {
	yearFolder := path.Join(imagesFolderPath, strconv.Itoa(fileDate.Year()))
	folderName := fileDate.Format("2006-01-02")
	folderPath := path.Join(yearFolder, folderName)
	return folderPath
}
