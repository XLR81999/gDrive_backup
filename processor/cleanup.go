package processor

import (
	"archive/zip"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	channels "gDriveBackup/channels"
)

func ExitHandler(ctx context.Context, zw *zip.Writer, zipFile *os.File) {
	//Clean up function
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	for{
		select{
			case<-exit:
				log.Print("Got interrupt signal, closing tasks")
				close(channels.FolderChannel)
				close(channels.FileChannel)
				time.Sleep(15 * time.Second)
				Cleanup(zw, zipFile)
		}
	}
}

func Cleanup(zw *zip.Writer, zipFile *os.File) {
	// Close the zip writer and file only if there are no errors
	closeZip(zw, zipFile)

	log.Printf("Download Completed. Backup file created: %s\n", zipFile.Name())
	log.Printf("Closing....")
	time.Sleep(5 * time.Second)
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