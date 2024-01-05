package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"inxo.ru/sync/sync"
	"log"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("S3 Sync App")
	myWindow.Resize(fyne.Size{
		Width:  800,
		Height: 600,
	})

	// S3 Settings Form
	bucketEntry := widget.NewEntry()
	regionEntry := widget.NewEntry()
	idEntry := widget.NewEntry()
	tokenEntry := widget.NewEntry()
	endpointEntry := widget.NewEntry()
	localPathEntry := widget.NewEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Bucket", Widget: bucketEntry},
			{Text: "Endpoint", Widget: endpointEntry},
			{Text: "Region", Widget: regionEntry},
			{Text: "Id", Widget: idEntry},
			{Text: "Token", Widget: tokenEntry},
			{Text: "Local Path", Widget: localPathEntry},
		},
		OnSubmit: func() {
			// Handle form submission
			bucket := bucketEntry.Text
			region := regionEntry.Text
			token := tokenEntry.Text
			id := idEntry.Text
			endpoint := endpointEntry.Text
			localPath := localPathEntry.Text

			// Perform synchronization using the provided S3 settings and local path
			syncData(myWindow, bucket, endpoint, region, id, token, localPath)
		},
	}

	// Combine forms into a tab container
	tabs := container.NewAppTabs(
		container.NewTabItem("S3 Settings", container.New(layout.NewVBoxLayout(), form)),
	)

	// Set the icon for the application (optional)
	myWindow.SetIcon(theme.FyneLogo())

	myWindow.SetContent(container.NewVBox(
		container.NewHBox(widget.NewLabel("S3 Sync App")),
		tabs,
	))

	myWindow.ShowAndRun()
}

func syncData(myWindow fyne.Window, bucket string, endpoint string, region string, id string, token string, localPath string) {
	// Implement your synchronization logic here
	fmt.Printf("Syncing data with S3 settings - Bucket: %s, Region: %s, Token: %s\n", bucket, region, token)
	fmt.Printf("Syncing data with S3 settings - Endpoint: %s, Id: %s\n", endpoint, id)
	fmt.Printf("Syncing data with local path: %s\n", localPath)

	// You can replace this with your actual synchronization logic
	if _, err := os.Stat(localPath); err == nil {
		sync.Sync()
		dialog.ShowInformation("Sync Success", "Data synchronized successfully!", myWindow)
	} else {
		log.Println("Error syncing data:", err)
		dialog.ShowError(err, myWindow)
	}
}
