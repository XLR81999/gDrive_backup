package processor

import (
	"archive/zip"
	"fmt"
	"log"

	channels "gDriveBackup/channels"

	"google.golang.org/api/drive/v3"
)

func ProcessFolder(service *drive.Service, folderMap map[string]string, zw *zip.Writer){
	folderID := folderMap["id"]
	path := folderMap["path"]
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)
	files, err := service.Files.List().
		Q(query).
		Fields("nextPageToken, files(id, name, mimeType)").
		Do()
	if err != nil {
		log.Printf("Error listing files in folder %s: %v\n", folderID, err)
		return
	}

	fmt.Printf("Found %d objects in folder %s\n", len(files.Files), folderID)

	for _, file := range files.Files {
		var filePath string
		if len(path) == 0 {
			filePath = file.Name
		} else {
			filePath = path + "/" + file.Name
		}
		if file.MimeType == "application/vnd.google-apps.folder" {
			folderMap := make(map[string]string)
			folderMap["id"] = file.Id
			folderMap["path"] = filePath
			channels.FolderChannel <- folderMap
		} else {
			fileMap := make(map[string]string)
			fileMap["id"] = file.Id
			fileMap["name"] = file.Name
			
			fileMap["path"] = filePath

			channels.FileChannel <- fileMap
		}
	}
}