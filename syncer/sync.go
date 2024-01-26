package syncer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"fyne.io/fyne/v2/widget"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"inxo.ru/sync/functions"
	"inxo.ru/sync/utils"
	"io"
	"path/filepath"
	"strings"

	"log"
	"os"
)

type Sync struct {
	Progress   *widget.ProgressBarInfinite
	LocalPath  string
	IgnoreDots bool
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
	s.IgnoreDots = false
	ignoreDotsEnv := os.Getenv("SYNC_IGNORE_DOTS")
	if ignoreDotsEnv == "true" {
		s.IgnoreDots = true
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
	if s.isLocalDirEmpty() {
		// Download files from S3 if the local directory is empty
		err := s.downloadFromS3(svc, bucketName)
		if err != nil {
			return err
		}
	} else {
		// Walk through the local folder and sync files
		err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			fileName := filepath.Base(path)
			if s.IgnoreDots && strings.HasPrefix(fileName, ".") {
				return nil
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Get md5 hash
			md5Hash, err := s.calculateLocalMD5(path)

			// Get relative path
			relPath, err := filepath.Rel(localPath, path)
			if err != nil {
				return err
			}

			// Check if object already exists in the bucket
			if _, ok := existingObjects[relPath]; ok {
				// Get hash
				md5HashS3, err := s.getObjectMetadata(svc, bucketName, relPath)
				if err != nil {
					return err
				}
				if md5HashS3 == md5Hash {
					log.Printf("Skipping existing object: %s", relPath)
					delete(existingObjects, relPath)
					return nil
				}
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

			if err != nil {
				return err
			}

			uploadObjectInput := &s3.PutObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(relPath),
				Body:   file,
				ACL:    aws.String("private"),
				Metadata: map[string]*string{
					"md5-hash": aws.String(md5Hash),
				},
			}

			_, err = svc.PutObject(uploadObjectInput)
			if err != nil {
				return err
			}

			log.Printf("Uploaded object: %s", relPath)
			return nil
		})
	}
	for relPath := range existingObjects {
		fileName := filepath.Base(relPath)
		if s.IgnoreDots && strings.HasPrefix(fileName, ".") {
			return nil
		}
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
func (s *Sync) calculateLocalMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	hash := md5.New()
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	hashInBytes := hash.Sum(nil)
	md5Hash := hex.EncodeToString(hashInBytes)
	return md5Hash, nil
}
func (s *Sync) isLocalDirEmpty() bool {
	dir, err := os.Open(s.LocalPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(dir *os.File) {
		err := dir.Close()
		if err != nil {

		}
	}(dir)

	_, err = dir.Readdirnames(1)
	return err == io.EOF
}

func (s *Sync) getObjectMetadata(client *s3.S3, bucket string, objectKey string) (string, error) {
	// Получаем метаданные объекта
	headObjectOutput, err := client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return "", err
	}

	// Извлекаем MD5 хеш из пользовательских метаданных
	md5HashFromS3 := strings.ReplaceAll(aws.StringValue(headObjectOutput.ETag), "\"", "")
	//if md5HashFromS3 == nil {
	//	return "", nil
	//}
	return md5HashFromS3, nil
}

func (s *Sync) downloadFromS3(client *s3.S3, bucket string) error {
	// List objects in the S3 bucket
	listObjectsV2Input := s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	objectCh, err := client.ListObjectsV2(&listObjectsV2Input)
	functions.CheckErr(err)

	for _, object := range objectCh.Contents {
		// Download the object from S3
		objectName := object.Key
		fileName := filepath.Base(*objectName)
		if s.IgnoreDots && strings.HasPrefix(fileName, ".") {
			return nil
		}
		objectPath := filepath.Join(s.LocalPath, *objectName)
		getInputObject := &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(*objectName),
		}
		objectOutput, err := client.GetObject(getInputObject)
		if err != nil {
			return err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				return
			}
		}(objectOutput.Body)
		err = os.MkdirAll(filepath.Dir(objectPath), 0755)
		if err != nil {
			return err
		}

		file, err := os.Create(objectPath)
		if err != nil {
			return err
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				return
			}
		}(file)

		_, err = file.ReadFrom(objectOutput.Body)
		if err != nil {
			return err
		}

		fmt.Printf("Downloaded file from S3: %s\n", objectPath)
	}

	return nil
}
