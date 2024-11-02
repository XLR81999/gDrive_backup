package processor

import (
	"archive/zip"
	"log"

	channels "gDriveBackup/channels"

	"google.golang.org/api/drive/v3"
)

func ProcessFolder(service *drive.Service, folderMap map[string]string, zw *zip.Writer){
	folderID := folderMap["id"]
	path := folderMap["path"]
	query := "'" + folderID + "' in parents"
	pageToken := ""
	for {
		files, err := service.Files.List().
			Fields("nextPageToken, files/*").
			PageToken(pageToken).
			PageSize(10).
			OrderBy("folder, modifiedTime desc, name").
			Q(query).
			Do()

		// fmt.Printf("Folder Data: %v\n", files)
		
		if err != nil {
			log.Printf("Error listing files in folder %s: %v\n", folderID, err)
			return
		}

		log.Printf("Found %v objects in folder %s", len(files.Files), path)

		for _, file := range files.Files {
			// log.Printf("Folder with name %s has parent %s\n", file.Name, file.Parents)
			var filePath string
			if len(path) == 0 {
				filePath = file.Name
			} else {
				filePath = path + "/" + file.Name
			}
			// log.Printf("Object with name: %s is of type: %s", file.Name, file.MimeType)
			if file.MimeType == "application/vnd.google-apps.folder"  {
				folderMap := make(map[string]string)
				folderMap["id"] = file.Id
				folderMap["path"] = filePath
				channels.FolderChannel <- folderMap
			} else if  file.MimeType != "application/vnd.google-apps.shortcut"{
				// log.Printf("File: %v\n", file)
				fileMap := make(map[string]string)
				fileMap["id"] = file.Id
				fileMap["name"] = file.Name
				fileMap["path"] = filePath
				fileMap["mimeType"] = file.MimeType
				channels.FileChannel <- fileMap
			}
		}
		pageToken = files.NextPageToken
		if pageToken == ""{
			break
		}
	}

	// log.Printf("Folder channel has lag of %v", len(channels.FolderChannel))
	// log.Printf("File channel has lag of %v", len(channels.FileChannel))
}