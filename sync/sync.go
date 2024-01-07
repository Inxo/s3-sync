package sync

import (
	"fyne.io/fyne/v2/widget"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"inxo.ru/sync/utils"
	"path/filepath"

	"log"
	"os"
)

type Sync struct {
	Progress  *widget.ProgressBarInfinite
	LocalPath string
}

func (s *Sync) Do() error {
	wd, err := os.Getwd()

	if s.Progress != nil {
		s.Progress.Show()
	}
	if err != nil {
		log.Fatal("Cannot get working directory")
	}
	logfile := utils.InitLogger(wd)
	if logfile == nil {
		log.Fatal("Cannot make log")
	}
	defer func(logfile *os.File) {
		err := logfile.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}(logfile)

	err = godotenv.Overload(wd + "/.env")
	if err != nil {
		return err
	}
	// Load environment variables
	bucketName := os.Getenv("BUCKET_NAME")
	localPath := s.LocalPath
	if localPath == "" {
		localPath = os.Getenv("LOCAL_PATH")
	}

	// Validate environment variables
	if bucketName == "" || localPath == "" {
		log.Fatal(".env file not loaded or missing required variables")
		return err
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("AWS_ENDPOINT")),
		Region:           aws.String(os.Getenv("AWS_REGION")),
		S3ForcePathStyle: aws.Bool(true),
	}

	// Create a new AWS session
	sess, err := session.NewSession(s3Config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new S3 client
	svc := s3.New(sess)

	// List objects in the bucket
	listObjectsInput := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	}

	listObjectsOutput, err := svc.ListObjects(listObjectsInput)
	if err != nil {
		return err
	}
	// Create a map to store existing objects in the bucket
	existingObjects := make(map[string]bool)
	for _, object := range listObjectsOutput.Contents {
		existingObjects[*object.Key] = true
	}

	// Walk through the local folder and sync files
	err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// Check if object already exists in the bucket
		if _, ok := existingObjects[relPath]; ok {
			log.Printf("Skipping existing object: %s", relPath)
			delete(existingObjects, relPath)
			return nil
		}

		// Upload file to S3
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Println(err)
			}
		}(file)

		uploadObjectInput := &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(relPath),
			Body:   file,
			ACL:    aws.String("private"),
		}

		_, err = svc.PutObject(uploadObjectInput)
		if err != nil {
			return err
		}

		log.Printf("Uploaded object: %s", relPath)
		return nil
	})
	for relPath := range existingObjects {
		// remove from s3
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(relPath),
		}
		_, err = svc.DeleteObject(deleteObjectInput)
		if err != nil {
			return err
		}
		log.Printf("Deleted object: %s", relPath)
	}
	if s.Progress != nil {
		s.Progress.Stop()
		s.Progress.Hide()
	}

	if err != nil {
		return err
	}

	log.Println("Sync completed!")
	return nil
}
