package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Velocidex/ordereddict"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"www.velocidex.com/golang/go-ese/parser"
)

var (
	dump_command = app.Command(
		"dump", "Dump table.")

	dump_command_file_arg = dump_command.Arg(
		"file", "The image file to inspect",
	).Required().OpenFile(os.O_RDONLY, os.FileMode(0666))

	dump_command_table_name = dump_command.Arg(
		"table", "The name of the table to dump").
		Required().String()
)

func doDump() {
	s, err := (*dump_command_file_arg).Stat()
	kingpin.FatalIfError(err, "Unable to open ese file")

	ese_ctx, err := parser.NewESEContext(*dump_command_file_arg, s.Size())
	kingpin.FatalIfError(err, "Unable to open ese file")

	catalog, err := parser.ReadCatalog(ese_ctx)
	kingpin.FatalIfError(err, "Unable to open ese file")

	err = catalog.DumpTable(*dump_command_table_name, func(row *ordereddict.Dict) error {
		serialized, err := json.Marshal(row)
		if err != nil {
			return err
		}
		fmt.Printf("%v\n", string(serialized))

		return nil
	})
	kingpin.FatalIfError(err, "Unable to open ese file")
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case dump_command.FullCommand():
			doDump()
		default:
			return false
		}
		return true
	})
}
