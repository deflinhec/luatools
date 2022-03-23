package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	luatools "github.com/deflinhec/luatools"
	"github.com/jessevdk/go-flags"
)

var (
	Version = "0.0.0"
	Build   = "-"
)

var opts struct {
	LuaFile string `long:"file" short:"f" description:"Lua 檔案路徑" default:"RoomData.lua"`

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
	f := luatools.NewDataPrettify()
	if err := f.Load(opts.LuaFile); err != nil {
		log.Fatalf("載入失敗 %v", err)
	}
	if abspath, err := filepath.Abs(opts.LuaFile); err == nil {
		opts.LuaFile = abspath
	}
	log.Printf("檔案路徑: %v", opts.LuaFile)
	os.Mkdir("output", os.ModePerm)
	ext := filepath.Ext(opts.LuaFile)
	dir := filepath.Dir(opts.LuaFile)
	name := filepath.Base(opts.LuaFile)
	field := regexp.MustCompile("[^A-Z|a-z]").
		ReplaceAllString(strings.ReplaceAll(name, ext, ""), "")
	log.Printf("輸出 %v ...", field)
	path := filepath.Join(dir, "output", name)
	if err := f.WriteToFile(path); err != nil {
		log.Fatalf("檔案輸出失敗 %v", err)
	}
	log.Printf("輸出路徑: %v", path)
}
