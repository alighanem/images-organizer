# Images organizer

This script organizes all files by year/date folder.

Each file will be moved to the appropriate folder.
The date folder is based on the image taken date in the Exif metadata (if not found, it is the modification date).
If the file is not an image (a video for example), the modification date is read only.

If the image has been taken the 2022-02-18, it will be moved to
2022/2022-02-18 folder.