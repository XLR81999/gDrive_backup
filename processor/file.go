package processor

import (
	"archive/zip"
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
	mimeType := fileMap["mimeType"]
	// fmt.Printf("Downloading file %s\n", fileName)
	fileContent, err := service.Files.Get(fileID).Download()
	if err != nil {
		// log.Printf("Got File %s with type %s", fileName, mimeType)
		downloadableMimeType, extension := getMimeType(mimeType)
		filePath += extension
		fileContent, err = service.Files.Export(fileID, downloadableMimeType).Download()
		if err != nil {
			log.Printf("Error downloading file %s with error %v\n, source type: %s, target type:  %s", fileName, err, mimeType, downloadableMimeType)
			return
		}

	}
	// log.Printf("Downloaded File %s", fileName)

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
	// log.Printf("File: %s has been copied to zip folder", fileName)
}

func getMimeType(mimeType string) (string, string) {
	if mimeType == "application/vnd.google-apps.spreadsheet" {
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"
	} else if mimeType == "application/vnd.google-apps.document" {
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".docx"
	} else if mimeType == "application/vnd.google-apps.presentation" {
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation", ".pptx"
	} else if mimeType == "application/vnd.google-apps.jam" {
		return "application/pdf", ".pdf"
	} else if mimeType == "application/vnd.google-apps.form" {
		return "application/zip", ".zip"
	}
	return mimeType, ".pdf"
}
