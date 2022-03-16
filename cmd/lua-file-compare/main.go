package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"future.net.co.luatools/pkg/luatools"
	"github.com/jessevdk/go-flags"
)

var (
	Version = "0.0.0"
	Build   = "-"
)

var opts struct {
	LuaFiles []string `long:"file" short:"f" description:"Lua A/B 比對檔案路徑" default:"RoomData.lua"`

	Version func() `long:"version" short:"v" description:"檢視建置版號"`
}

var parser = flags.NewParser(&opts, flags.Default)

func init() {
	opts.Version = func() {
		fmt.Printf("Version: %v", Version)
		fmt.Printf("\tBuild: %v", Build)
		os.Exit(0)
	}
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
}

func main() {
	cmp := luatools.NewDataComparism()
	files := opts.LuaFiles
	if len(files) < 2 {
		log.Fatal("無法進行比對")
	} else if len(files) > 2 {
		log.Print("一次只能比對兩個檔案")
	}
	if err := cmp.Load(files); err != nil {
		log.Fatal(err)
	}
	for i, file := range files {
		if abspath, err := filepath.Abs(file); err == nil {
			file = abspath
		}
		log.Printf("比對檔案: %v %v", cmp.Checksums[i], file)
	}
	name := filepath.Base(opts.LuaFiles[0])
	name = strings.ReplaceAll(name, filepath.Ext(name), "")
	name = regexp.MustCompile("[^A-Z|a-z]").ReplaceAllString(name, "")
	log.Printf("比對 %v ...", name)
	ret, err := cmp.Equal(name)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("比對結果: %v", ret)
}
