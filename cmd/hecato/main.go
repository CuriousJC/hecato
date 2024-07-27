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
	"strconv"

	"github.com/curiousjc/hecato/internal/listfiles"
	"github.com/curiousjc/hecato/internal/version"
)

func main() {
	fmt.Println("-----------------Running Hecato-------------")

	method := flag.String("method", "version", "Specify the method to run")
	target := flag.String("target", "undefined", "target of the given method")
	hits := flag.String("hits", "15", "How many hits for given method")

	flag.Parse()

	//everything begins with the method
	switch *method {
	case "version":
		version.Print()
	case "largefiles":
		files, err := listfiles.GetLargeFiles(*target, *hits)
		if err != nil {
			fmt.Println("Error listing files: ", err)
		}

		for i, file := range files {
			fmt.Printf(" %s. Size: %s bytes | Path: %s \n", strconv.Itoa(i), file.SizeInMB(), file.Path)
		}
	case "examples":
		fmt.Println("go run c:/repos/hecato/cmd/hecato/main.go -method largefiles -hits 25 -target d:/SteamLibrary")

	default:
		fmt.Println("That wasn't an option for our 'm' method")
	}
}
