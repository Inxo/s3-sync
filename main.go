package main

import (
	"bufio"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/joho/godotenv"
	"inxo.ru/sync/syncer"
	"inxo.ru/sync/utils"
	"log"
	"os"
)

func main() {

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln("No work dir")
	}
	// Load environment variables
	err = godotenv.Load(wd + "/.env")
	var localPath = ""
	if err != nil {
		localPath = wd
	} else {
		localPath = os.Getenv("LOCAL_PATH")
	}
	args := os.Args
	if len(args) >= 2 && args[1] != "" {
		localPath = args[1]
		s := syncer.Sync{
			Progress:  nil,
			LocalPath: localPath,
		}
		err := s.Do()
		if err != nil {
			return
		}
		fmt.Println("Synced: " + localPath)
		return
	}

	myApp := app.NewWithID("com.n32b.capysync")

	// S3 Settings Form
	bucketEntry := widget.NewEntry()
	bucketEntry.SetText(os.Getenv("BUCKET_NAME"))
	regionEntry := widget.NewEntry()
	regionEntry.SetText(os.Getenv("AWS_REGION"))
	idEntry := widget.NewEntry()
	idEntry.SetText(os.Getenv("AWS_ACCESS_KEY_ID"))
	tokenEntry := widget.NewEntry()
	tokenEntry.SetText(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	endpointEntry := widget.NewEntry()
	endpointEntry.SetText(os.Getenv("AWS_ENDPOINT"))
	localPathEntry := widget.NewEntry()
	localPathEntry.SetText(localPath)

	progressEntry := widget.NewProgressBarInfinite()
	progressEntry.Hide()
	myWindow := myApp.NewWindow("Capy Sync")

	saveShort := func() {
		// Handle form submission
		bucket := bucketEntry.Text
		region := regionEntry.Text
		token := tokenEntry.Text
		id := idEntry.Text
		endpoint := endpointEntry.Text
		localPath := localPathEntry.Text

		// Perform save data
		saveData(myWindow, bucket, endpoint, region, id, token, localPath, wd)
	}

	syncShort := func() {
		// Handle form submission
		bucket := bucketEntry.Text
		region := regionEntry.Text
		token := tokenEntry.Text
		id := idEntry.Text
		endpoint := endpointEntry.Text
		localPath := localPathEntry.Text
		progress := progressEntry

		// Perform synchronization using the provided S3 settings and local path
		syncData(myWindow, progress, bucket, endpoint, region, id, token, localPath)
	}
	if desk, ok := myApp.(desktop.App); ok {
		m := fyne.NewMenu("MyApp",
			fyne.NewMenuItem("Show", func() {
				myWindow.Show()
			}),
			fyne.NewMenuItem("Sync", syncShort))
		desk.SetSystemTrayMenu(m)
	}

	myWindow.Resize(fyne.Size{
		Width:  800,
		Height: 600,
	})
	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Bucket", Widget: bucketEntry},
			{Text: "Endpoint", Widget: endpointEntry},
			{Text: "Region", Widget: regionEntry},
			{Text: "Id", Widget: idEntry},
			{Text: "Token", Widget: tokenEntry},
			{Text: "Local Path", Widget: localPathEntry},
		},
		OnSubmit:   saveShort,
		SubmitText: "Save",
	}

	syncForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Progress", Widget: progressEntry},
		},
		OnSubmit:   syncShort,
		SubmitText: "Sync",
	}

	// Combine forms into a tab container
	tabs := container.NewAppTabs(
		container.NewTabItem("Sync", container.New(layout.NewVBoxLayout(), syncForm)),
		container.NewTabItem("Settings", container.New(layout.NewVBoxLayout(), form)),
	)

	// Set the icon for the application (optional)
	myWindow.SetIcon(theme.AccountIcon())

	myWindow.SetContent(container.NewVBox(
		container.NewHBox(widget.NewLabel("Sync App - free and open source")),
		tabs,
	))

	myWindow.ShowAndRun()
}

func saveData(myWindow fyne.Window, bucket string, endpoint string, region string, id string, token string, path string, wd string) {
	file, err := os.Create(wd + "/.env")
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	writer := bufio.NewWriter(file)
	_, _ = fmt.Fprintf(writer, "LOCAL_PATH=%s\n", path)
	_, _ = fmt.Fprintf(writer, "BUCKET_NAME=%s\n", bucket)
	_, _ = fmt.Fprintf(writer, "AWS_ACCESS_KEY_ID=%s\n", id)
	_, _ = fmt.Fprintf(writer, "AWS_ENDPOINT=%s\n", endpoint)
	_, _ = fmt.Fprintf(writer, "AWS_SECRET_ACCESS_KEY=%s\n", token)
	_, _ = fmt.Fprintf(writer, "AWS_REGION=%s\n", region)
	err = writer.Flush()
	if err != nil {
		dialog.ShowError(err, myWindow)
	} else {
		dialog.ShowInformation("Save Success", "Data save successfully!", myWindow)
	}
}

func syncData(myWindow fyne.Window, progress *widget.ProgressBarInfinite, bucket string, endpoint string, region string, id string, token string, localPath string) {
	// Implement your synchronization logic here
	if utils.IsDebug() {
		fmt.Printf("Syncing data with S3 settings - Bucket: %s, Region: %s, Token: %s\n", bucket, region, token)
		fmt.Printf("Syncing data with S3 settings - Endpoint: %s, Id: %s\n", endpoint, id)
		fmt.Printf("Syncing data with local path: %s\n", localPath)
	}

	// You can replace this with your actual synchronization logic
	if _, err := os.Stat(localPath); err == nil {
		s := syncer.Sync{
			Progress:  progress,
			LocalPath: localPath,
		}
		err2 := s.Do()
		if err2 != nil {
			e := errors.New(err2.Error())
			s.Progress.Stop()
			dialog.ShowError(e, myWindow)

		} else {
			dialog.ShowInformation("Sync Success", "Data synchronized successfully!", myWindow)
		}
	} else {
		log.Println("Error syncing data:", err)
		dialog.ShowError(err, myWindow)
	}
}
