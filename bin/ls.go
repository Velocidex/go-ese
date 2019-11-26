package main

import (
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"www.velocidex.com/golang/go-ese/parser"
)

var (
	ls_command = app.Command(
		"ls", "List files.")

	ls_command_file_arg = ls_command.Arg(
		"file", "The image file to inspect",
	).Required().OpenFile(os.O_RDONLY, os.FileMode(0666))
)

func doLS() {
	s, err := (*ls_command_file_arg).Stat()
	kingpin.FatalIfError(err, "Unable to open ese file")

	ese_ctx, err := parser.NewESEContext(*ls_command_file_arg, s.Size())
	kingpin.FatalIfError(err, "Unable to open ese file")

	fmt.Printf("FileHeader: %v\n", ese_ctx.Header.DebugString())

	catalog := parser.ReadCatalog(ese_ctx)
	catalog.Dump()

	catalog.DumpTable("MSysObjects")
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case "ls":
			doLS()
		default:
			return false
		}
		return true
	})
}
