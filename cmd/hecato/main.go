/*

main executable for hecato

go run c:/repos/hecato/cmd/hecato/main.go
go build -o hecato.exe c:/repos/hecato/cmd/hecato/main.go
c:/repos/hecato/hecato.exe

go run c:/repos/hecato/cmd/hecato/main.go -method largefiles -hits 10 -target 'c:/windows' -hide=false

git tag v1.0.0
git push origin v1.0.0
git push origin main --tags


*/

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
	hide := flag.Bool("hide", true, "Hide access errors from output")
	examples := flag.Bool("examples", false, "Show examples of usage")

	flag.Parse()

	if *examples {
		*method = "examples"
	}

	//everything begins with the method
	//TODO: last changed files, last created files, file contents search, directory size summary
	switch *method {
	case "version":
		version.Print()
	case "largefiles":
		foundFiles, errorFiles, err := listfiles.GetLargeFiles(*target, *hits)
		if err != nil {
			fmt.Println("Error listing files: ", err)
		}

		for i, file := range foundFiles {
			fmt.Printf(" %s. Size: %s bytes | Path: %s \n", strconv.Itoa(i), file.SizeInMB(), file.Path)
		}
		if !*hide {
			for _, file := range errorFiles {
				fmt.Printf(" Following access errors encountered: %s \n", file.Path)
			}
		}
	case "examples":
		fmt.Println("----Examples of usage-----")
		fmt.Println("hecato -method largefiles -hits 25 -target d:/SteamLibrary")
		fmt.Println("hecato -method largefiles -hits 10 -target 'c:/windows' -hide=false")

	default:
		fmt.Println("That wasn't an option for our 'm' method")
	}
}
