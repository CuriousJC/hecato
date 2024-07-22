/*

main executable for hecato

go run c:/repos/hecato/cmd/hecato/main.go
go build -o hecato.exe c:/repos/hecato/cmd/hecato/main.go
c:/repos/hecato/hecato.exe

*/

//TODO: make the name of the executable 'hecato'

package main

import (
	"flag"
	"fmt"
	"github.com/curiousjc/hecato/internal/listfiles"
	"github.com/curiousjc/hecato/internal/version"
)

func main() {
	fmt.Println("-----------------Running Hecato-------------")

	method := flag.String("m", "version", "Specify the method to run")
	//TODO: add a help argument (and a -h argument)
	//TODO: add a "hit" argument for the number of hits to return
	//TODO: add a "-method" argument
	flag.Parse()

	switch *method {
	case "version":
		version.Print()
	case "largefiles":
		files, err := listfiles.GetLargeFiles("D:/")
		if err != nil {
			fmt.Println("Error listing files: ", err)
		}

		for _, file := range files {
			fmt.Printf("Size: %s bytes | Path: %s \n", file.SizeInMB(), file.Path)
		}

	default:
		fmt.Println("That wasn't an option for our 'm' method")
	}
}
