package main

import (
	"archive/zip"
	"fmt"
	"time"

	"log"
	"os"

	channels "gDriveBackup/channels"
	processor "gDriveBackup/processor"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)



func main() {
	// Path to Service Account Key
	serviceAccountKey := "C:\\Users\\sridh\\Downloads\\creds.json"
	ctx := context.Background()

	// Get the token
	serviceAccountKeyBytes, err := os.ReadFile(serviceAccountKey)

	if err != nil {
		log.Fatalf("Error getting Service Account Bytes: %v", err)
	}
	scopes := []string{"https://www.googleapis.com/auth/drive.readonly"}
	jwtToken, err := google.JWTConfigFromJSON(serviceAccountKeyBytes, scopes...)
	if err != nil {
		log.Fatalf("Error getting JWT token: %v", err)
	}

	tokenSource := jwtToken.TokenSource(ctx)
	// Create a new OAuth client
	client := oauth2.NewClient(ctx, tokenSource)

	// Create a new Drive service
	service, err := drive.New(client)
	if err != nil {
		log.Fatalf("Error creating Drive service: %v", err)
	}

	_, err = service.About.Get().Fields("*").Do()
	if err != nil {
		log.Fatalf("Error validating API Key: %v", err)
	}

	var query string
	folderID := "1GQgtVXkJM5q6GywHhD6x4cALNtLu81-q"
	if folderID != "" {
		query = fmt.Sprintf("'%s' in parents\n", folderID)
		fmt.Print(query)
	} else {
		log.Fatalf("No folder ID given")
		os.Exit(1)
		query = "" // Retrieve all files if folder ID is not provided
	}

	// Create a zip file
	zipFile, err := os.Create("backup.zip")
	if err != nil {
		log.Fatalf("Error creating zip file: %v", err)
	}
	zw := zip.NewWriter(zipFile)
	// Start the recursive download
	channels.FolderChannel = make(chan map[string]string, 10)
	channels.FileChannel = make(chan map[string]string, 100)
	err = downloadFiles(service, folderID, zw)
	if err != nil {
		log.Fatalf("Error downloading files: %v", err)
	}
	// Close the zip writer and file only if there are no errors
	closeZip(zw, zipFile)

	fmt.Printf("Download complete. Backup file created: %s\n", zipFile.Name())
}

func closeZip(zw *zip.Writer, zipFile *os.File) {
	err := zw.Close()
	if err != nil {
		log.Fatalf("Error closing zip writer: %v", err)
	}
	err = zipFile.Close()
	if err != nil {
		log.Fatalf("Error closing zip file: %v", err)
	}
}

func isProcessingDone() bool {
	if len(channels.FolderChannel) == 0 && len(channels.FileChannel) == 0{
		log.Print("Waiting for 30 sec to make sure no thread is in process of fetching more folders/files")
		time.Sleep(30 * time.Second)
		if len(channels.FolderChannel) == 0 && len(channels.FileChannel) == 0{
			return true
		}
	}
	return false
}

func downloadFiles(service *drive.Service, folderID string, zw *zip.Writer) error {
	channels.FolderChannel <- map[string]string{"id": folderID, "path": ""}
	channels.Wg.Add(1)

	for i := 0; i < 2; i++ {
		go func() {
			for folderID := range channels.FolderChannel {
				fmt.Printf("Processing folder ID %s\n", folderID)
				processor.ProcessFolder(service, folderID, zw)
			}
		}()
	}

	for j := 0; j < 2; j++ {
		go func() {
			for fileMap := range channels.FileChannel {
				fmt.Printf("Processing file ID %s\n", fileMap["id"])
				processor.ProcessFile(service, zw, fileMap)
			}
		}()
	}


	go func(){
		isDone := false
		for !isDone {
			log.Print("Checking if processing of files and folders are done")
			isDone = isProcessingDone()
			if isDone {
				channels.Wg.Done()
				return
			}
			log.Print("Sleeping for 0.5 min before checking again")
			time.Sleep(30 * time.Second)
		}
	}()

	channels.Wg.Wait()
	log.Println("All goroutines finished, closing channels")
	close(channels.FolderChannel)
	close(channels.FileChannel)

	return nil
}


