package examples

import (
	"fmt"
)

func Print() {
	fmt.Println("---- Examples ------")
	fmt.Println("hecato -m=largefiles -target=\"d:\" #show default 15 large files in D drive")
	fmt.Println("hecato -method=largefiles -hits=25 -target=d:/SteamLibrary #show 25 large files in the SteamLibrary directory")
	fmt.Println("hecato -method=largefiles -hits=10 -target=\"c:/windows\" -verbose=true #show 10 large files in windows including the errors that are usually hidden")
	fmt.Println("hecato -method=modfiles -hits=25 -target=\"c:/repos\" #show the 25 most recently modified files under c:/repos")
	fmt.Println("")
	fmt.Println("---- Config ------")
	fmt.Println("hecato -method=initconfig #write a starter hecato.yaml beside the executable")
	fmt.Println("hecato -method=initconfig -config=c:/tmp/hecato.yaml #write it somewhere specific instead")
	fmt.Println("hecato -method=largefiles -target=\"c:/\" -config=c:/tmp/hecato.yaml #use a specific config for this run")
	fmt.Println("")
	fmt.Println("A config is optional. Without one hecato behaves exactly as it always has.")
	fmt.Println("With one you can set default method/target/hits/verbose, skip OS files like")
	fmt.Println("pagefile.sys, and ignore paths. An ignore entry ending in / prunes that whole")
	fmt.Println("directory, so the walk never descends into it.")
}
