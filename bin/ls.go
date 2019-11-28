package main

import (
	"encoding/json"
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

	catalog := parser.ReadCatalog(ese_ctx)
	cursor, err := catalog.OpenTable(ese_ctx, "{5C8CF1C7-7257-4F13-B223-970EF5939312}")
	kingpin.FatalIfError(err, "Unable to open ese file")

	for row := cursor.GetNextRow(); row != nil; {
		serialized, err := json.Marshal(row)
		if err == nil {
			fmt.Printf("%v\n", string(serialized))
		}
	}
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
