package main

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

type CLI struct {
	Logger                *slog.Logger
	SourceFolderPath      string
	DestinationFolderPath string
	DryRun                bool

	ifdMapping *exifcommon.IfdMapping
}

var allowedExtensions = map[string]struct{}{
	".jpg": {},
	".png": {},
	".mp4": {},
	".avi": {},
}

func configure() *CLI {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	dryRunValue := os.Getenv(("DRY_RUN"))
	dryRun := true
	if dryRunValue != "" {
		v, err := strconv.ParseBool(dryRunValue)
		if err != nil {
			logger.Error("dry run value invalid", "dry_run", dryRunValue)
			return nil
		}
		dryRun = v
	}

	// This folder is the folder where are located the images we want to move
	sourceFolderPath := os.Getenv("PICTURES_FOLDER")
	if sourceFolderPath == "" {
		logger.Error("Source Images folder is not defined")
		return nil
	}

	destinationFolderPath := os.Getenv("DESTINATION_FOLDER")
	if sourceFolderPath == "" {
		logger.Error("Destination folder is not defined")
		return nil
	}

	// use path join to format correctly the source folder path
	sourceFolderPath = filepath.Join(sourceFolderPath)

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		logger.Error("init ifd mapping", "err", err)
		return nil
	}

	return &CLI{
		Logger:                logger,
		DryRun:                dryRun,
		SourceFolderPath:      sourceFolderPath,
		DestinationFolderPath: destinationFolderPath,
		ifdMapping:            im,
	}
}

func main() {
	c := configure()
	if c == nil {
		slog.Error("failed to configure cli")
		return
	}

	log := c.Logger

	files := make(map[string]fs.FileInfo)
	// Read the folder and subfolders to get all files
	err := filepath.WalkDir(c.SourceFolderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		extension := strings.ToLower(filepath.Ext(path))
		if _, allowed := allowedExtensions[extension]; !allowed {
			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		files[path] = info

		return nil
	})

	if err != nil {
		log.Error("failed reading folder", "folder_path", c.SourceFolderPath, "err", err)
	}

	if len(files) == 0 {
		log.Info("no files found in the folder")
		return
	}

	log.Info("path found", "count", len(files))

	i := 0

	for oldPath, file := range files {
		// Compute the destination folder of the image based on its modification date
		// The destination folder will be like C:/images/2024/2024-06-15
		takenTime, err := c.getTakenTime(oldPath, file)
		if err != nil {
			log.Error("failed to get taken time", "path", oldPath, "err", err)
			continue
		}

		// Compute destination folder and the complete new path of the file
		destinationFolder := c.computeDestinationFolder(takenTime)
		newPath := filepath.Join(destinationFolder, file.Name())

		exist, err := pathExists(newPath)
		if err != nil {
			log.Error("checking file path", "err", err, "file_path", newPath)
			return
		}

		if exist {
			log.Error("cannot move file: already exists", "file_path", newPath)
			return
		}

		if c.DryRun {
			log.Warn("dry run", "old_path", oldPath, "new_path", newPath, "date", takenTime)
			continue
		}

		exist, err = pathExists(destinationFolder)
		if err != nil {
			log.Error("checking destination folder", "err", err, "folder_path", destinationFolder)
			return
		}

		if !exist {
			// Create the destination folder if it does not existy
			err = os.Mkdir(destinationFolder, 0755)
			if err != nil {
				log.Error("creating image folder", "folder_path", destinationFolder)
				return
			}
		}

		err = os.Rename(oldPath, newPath)
		if err != nil {
			log.Error("cannot move file", "name", file.Name(), "old path", oldPath, "new path", newPath, "err", err)
			continue
		}

		// Sleep to pause the system
		if i%5 == 0 {
			time.Sleep(100 * time.Millisecond)
		}

		i++
	}

	log.Info("finished")
}

// pathExists returns whether the given file or directory exists
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func (c *CLI) computeDestinationFolder(fileDate time.Time) string {
	yearFolder := filepath.Join(c.DestinationFolderPath, strconv.Itoa(fileDate.Year()))
	folderName := fileDate.Format("2006-01-02")
	folderPath := filepath.Join(yearFolder, folderName)
	return folderPath
}

func (c *CLI) getTakenTime(filePath string, fileInfo fs.FileInfo) (time.Time, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	// TODO keep this code for now
	// We only have to pass the file header = first 261 bytes
	/*head := make([]byte, 261)
	f.Read(head)

	if !filetype.IsImage(head) {
		return fileInfo.ModTime(), nil
	}

	// reset the reader to allow exif read the file
	f.Seek(0, 0)*/

	exifDate, err := c.getExifDate(f)
	if err != nil {
		c.Logger.Debug("exit date not found", "file", filePath, "err", err)
		// return modification date if exif date not found
		return fileInfo.ModTime(), nil
	}

	return exifDate, nil
}

func (c *CLI) getExifDate(r io.Reader) (time.Time, error) {
	rawExif, err := exif.SearchAndExtractExifWithReader(r)
	if err != nil {
		return time.Time{}, err
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(c.ifdMapping, ti, rawExif)
	if err != nil {
		return time.Time{}, err
	}

	exifTree, ok := index.Lookup["IFD/Exif"]
	if !ok {
		return time.Time{}, errors.New("IFD/Exif mapping not found")
	}

	results, err := exifTree.FindTagWithName("DateTimeOriginal")
	if err != nil {
		return time.Time{}, err
	}

	if len(results) != 1 {
		return time.Time{}, errors.New("DateTimeOriginal invalid")
	}

	v, err := results[0].Value()
	if err != nil {
		return time.Time{}, err
	}

	date, err := time.Parse("2006:01:02 15:04:05", v.(string))
	if err != nil {
		return time.Time{}, err
	}

	return date, nil
}
