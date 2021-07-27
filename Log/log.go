package log

import (
	"fmt"
	"log"
)

const (
	RESET  = "\033[0m"
	GREEN  = "\033[32m"
	RED    = "\033[31m"
	YELLOW = "\033[33m"
	BLUE   = "\033[34m"
)

func SVC(format string, a ...interface{}) {
	f := "[SVC ] " + format
	fmt.Print(GREEN)
	log.Printf(f, a...)
	fmt.Print(RESET)
	return
}

func ERR(format string, a ...interface{}) {
	f := "[ERR ] " + format
	fmt.Print(RED)
	log.Printf(f, a...)
	fmt.Print(RESET)
	return
}

func DBG(format string, a ...interface{}) {
	f := "[DBG ] " + format
	fmt.Print(YELLOW)
	log.Printf(f, a...)
	fmt.Print(RESET)
	return
}

func INFO(format string, a ...interface{}) {
	f := "[INFO] " + format
	fmt.Print(BLUE)
	log.Printf(f, a...)
	fmt.Print(RESET)
}
