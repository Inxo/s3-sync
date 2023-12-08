package utils

import (
	"fmt"
	"inxo.ru/sync/functions"
	"log"
	"os"
)

func InitLogger(wd string) *os.File {
	logFile := wd + "/logs/app.log"
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
		err = os.MkdirAll(wd+"/logs/", 0755)
		functions.CheckErr(err)
		f, err = os.Create(logFile)
		functions.CheckErr(err)
	}
	log.SetOutput(f)
	return f
}
