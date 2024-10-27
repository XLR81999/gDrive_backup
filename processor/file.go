package processor

import (
	"archive/zip"
	"fmt"
	"io"
	"log"

	"google.golang.org/api/drive/v3"

	channels "gDriveBackup/channels"
)

func ProcessFile(service *drive.Service, zw *zip.Writer, fileMap map[string]string) {
	// Download the file
	fileName := fileMap["name"]
	fileID := fileMap["id"]
	filePath := fileMap["path"]
	fmt.Printf("Downloading file %s\n", fileName)
	fileContent, err := service.Files.Get(fileID).Download()
	if err != nil {
		log.Printf("Error downloading file %s: %v\n", fileName, err)
		return
	}

	// Add file to zip
	channels.Mu.Lock()
	f, err := zw.Create(filePath)
	if err != nil {
		log.Printf("Error adding file %s to zip: %v\n", fileName, err)
		channels.Mu.Unlock()
		return
	}
	_, err = io.Copy(f, fileContent.Body)
	if err != nil {
		log.Printf("Error writing file %s to zip: %v\n", fileName, err)
	}
	channels.Mu.Unlock()
}