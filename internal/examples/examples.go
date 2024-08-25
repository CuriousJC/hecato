package examples

import (
	"fmt"
)

func Print() {
	fmt.Println("---- Examples ------")
	fmt.Println("hecato -m=largefiles -target=\"d:\" #show default 15 large files in D drive")
	fmt.Println("hecato -method=largefiles -hits=25 -target=d:/SteamLibrary #show 25 large files in the SteamLibrary directory")
	fmt.Println("hecato -method=largefiles -hits=10 -target=\"c:/windows\" -hide=false #show 10 large files in windows including the errors that are usually hidden")
}
