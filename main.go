package main

import (
	"archive/zip"
	"context"
	"fmt"
	"time"

	"log"
	"os"

	channels "gDriveBackup/channels"
	client "gDriveBackup/client"
	processor "gDriveBackup/processor"

	"google.golang.org/api/drive/v3"
)

func main() {
	ctx := context.Background()
	service, err := client.FetchDriveService()

	if err != nil {
		log.Fatalf("Error creating Drive service: %v", err)
	}

	_, err = service.About.Get().Fields("*").Do()
	if err != nil {
		log.Fatalf("Error validating API Key: %v", err)
	}
	log.Printf("Enter Folder ID to start backup from, Type root for entire backup: \n")

	var folderID string
	fmt.Scan(&folderID)

	// Create a zip file and the zip writer
	zipFile, err := os.Create("backup.zip")
	if err != nil {
		log.Fatalf("Error creating zip file: %v", err)
	}
	zw := zip.NewWriter(zipFile)

	//Initialize channels
	channels.FolderChannel = make(chan map[string]string, 21)
	channels.FileChannel = make(chan map[string]string, 101)

	//Start Exit handler
	go processor.ExitHandler(ctx, zw, zipFile)

	// Start the recursive download
	err = downloadFiles(service, folderID, zw)
	processor.Cleanup(zw, zipFile)
	if err != nil {
		log.Fatalf("Error downloading files: %v", err)
	}
}

func isProcessingDone() bool {
	if len(channels.FolderChannel) == 0 && len(channels.FileChannel) == 0 {
		log.Print("Waiting for 30 sec to make sure no thread is in process of fetching more folders/files")
		time.Sleep(30 * time.Second)
		if len(channels.FolderChannel) == 0 && len(channels.FileChannel) == 0 {
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
			log.Printf("Exiting Thread %v of folder thread", i)
		}()
	}

	for j := 0; j < 4; j++ {
		go func() {
			for fileMap := range channels.FileChannel {
				fmt.Printf("Processing file with name %s\n", fileMap["name"])
				processor.ProcessFile(service, zw, fileMap)
			}
			log.Printf("Exiting Thread %v of file thread", j)
		}()
	}

	go func() {
		isDone := false
		time.Sleep(5*time.Second)
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
