package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/luevano/luaprovider"
)

const (
	filenameSDK       = "sdk.lua"
	filenameProvider  = "provider.lua"
	filenameLuarcJSON = ".luarc.json"
)

func main() {
	dir := flag.String("dir", ".", "output directory")

	generateSDK := flag.Bool("sdk", false, "generate sdk.lua file for language server")
	generateProvider := flag.Bool("provider", false, "generate provider.lua template")
	generateLuarc := flag.Bool("luarc", false, "generate .luarc.json file for language server configuration")

	flag.Parse()

	if !*generateSDK && !*generateProvider && !*generateLuarc {
		flag.Usage()
		return
	}

	if *generateSDK {
		err := os.WriteFile(
			filepath.Join(*dir, filenameSDK),
			[]byte(luaprovider.LuaDoc()),
			0644,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *generateProvider {
		err := os.WriteFile(
			filepath.Join(*dir, filenameProvider),
			[]byte(luaprovider.LuaTemplate()),
			0644,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *generateLuarc {
		err := os.WriteFile(
			filepath.Join(*dir, filenameLuarcJSON),
			[]byte(luaprovider.LuarcJSON()),
			0644,
		)
		if err != nil {
			log.Fatal(err)
		}
	}
}
