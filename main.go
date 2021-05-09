package main

import (
	"breSvc/api"
	"fmt"
	"os"
)

func main() {

	if len(os.Args) == 2 {
		api.Port = os.Args[1]
	} else {
		api.Port = "9001"
	}

	fmt.Println("Initializing System ....... please wait")

	api.HandleReq()

}
