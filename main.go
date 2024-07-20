package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/h2non/filetype"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
)

func init() {
	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)
}

func main() {

	dryRunValue := os.Getenv(("DRY_RUN"))
	dryRun := true
	if dryRunValue != "" {
		v, err := strconv.ParseBool(dryRunValue)
		if err != nil {
			log.Fatal("dry run value invalid", "dry_run", dryRunValue)
		}
		dryRun = v
	}

	// This folder is the folder where are located the images we want to move
	sourceFolderPath := os.Getenv("PICTURES_FOLDER")
	if sourceFolderPath == "" {
		log.Fatal("Source Images folder is not defrined")
	}

	// use path join to format correctly the source folder path
	sourceFolderPath = filepath.Join(sourceFolderPath)

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
	rootPath := filepath.Dir(sourceFolderPath)
	i := 0

	for _, file := range files {
		if file.Name() == "desktop.ini" {
			// TODO improve it by allowing only certain file types
			// ignore it
			continue
		}

		oldPath := filepath.Join(sourceFolderPath, file.Name())

		// Compute the destination folder of the image based on its modification date
		// The destination folder will be like C:/images/2024/2024-06-15
		takenTime, err := getTakenTime(oldPath, file)
		if err != nil {
			log.Println("failed to get taken time", "path", oldPath, "err", err)
			continue
		}

		// Compute destination folder and the complete new path of the file
		destinationFolder := computeDestinationFolder(rootPath, takenTime)
		newPath := filepath.Join(destinationFolder, file.Name())

		if dryRun {
			log.Println("dry run", "old_path", oldPath, "new_path", newPath, "date", takenTime)
			continue
		}

		_, err = os.Stat(destinationFolder)
		if os.IsNotExist(err) {
			// Create the destination folder if it does not existy
			err = os.Mkdir(destinationFolder, 0755)
			if err != nil {
				log.Fatalln("creating image folder", "folder_path", destinationFolder)
			}
		}

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
	yearFolder := filepath.Join(imagesFolderPath, strconv.Itoa(fileDate.Year()))
	folderName := fileDate.Format("2006-01-02")
	folderPath := filepath.Join(yearFolder, folderName)
	return folderPath
}

func getTakenTime(filePath string, fileInfo fs.FileInfo) (time.Time, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	f.Read(head)

	if !filetype.IsImage(head) {
		return fileInfo.ModTime(), nil
	}

	// reset the reader to allow exif read the file
	f.Seek(0, 0)

	x, err := exif.Decode(f)
	if err != nil {
		if err.Error() == "EOF" {
			// Cannot decode exit maybe does not exist in the file
			// Returns the mod type
			return fileInfo.ModTime(), nil
		}

		return time.Time{}, err
	}

	date, err := x.DateTime()
	if err != nil {
		return time.Time{}, err
	}

	if !date.IsZero() {
		return date, nil
	}

	return fileInfo.ModTime(), nil
}
