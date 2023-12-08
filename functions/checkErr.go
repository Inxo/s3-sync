package functions

import "log"

type CheckErrorInterface interface {
	CheckErr(err error)
}

func CheckErr(err error) {
	if err != nil {
		log.Print(err)
	}
}
