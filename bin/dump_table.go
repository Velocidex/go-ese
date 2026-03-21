package main

import (
	"encoding/json"
	"errors"
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
		Strings()

	dump_command_table_name_limit = dump_command.Flag(
		"limit", "Only dump this many rows").Int()

	dump_command_pages = dump_command.Flag(
		"pages", "Only show the pages").Bool()

	STOP_ERROR = errors.New("Stop")
)

func doDump() {
	stat, err := (*dump_command_file_arg).Stat()
	kingpin.FatalIfError(err, "Unable to open ese file")

	ese_ctx, err := parser.NewESEContext(
		*dump_command_file_arg, stat.Size())
	kingpin.FatalIfError(err, "Unable to open ese file")

	catalog, err := parser.ReadCatalog(ese_ctx)
	kingpin.FatalIfError(err, "Unable to open ese file")

	tables := *dump_command_table_name
	if len(tables) == 0 {
		tables = catalog.Tables.Keys()
	}

	for _, t := range tables {
		count := 0

		if *dump_command_pages {
			table_any, pres := catalog.Tables.Get(t)
			if !pres {
				continue
			}

			table := table_any.(*parser.Table)
			err := parser.WalkPages(ese_ctx, int64(table.FatherDataPageNumber),
				func(header *parser.PageHeader, id int64, value *parser.Value) error {
					fmt.Printf("Page %v: %v\n", id, header.DebugString())
					return nil
				})
			kingpin.FatalIfError(err, "Unable to walk table pages")
			continue
		}

		err = catalog.DumpTable(t, func(row *ordereddict.Dict) error {
			serialized, err := json.Marshal(row)
			if err != nil {
				return err
			}

			count++
			fmt.Printf("%v\n", string(serialized))
			if *dump_command_table_name_limit > 0 &&
				count >= *dump_command_table_name_limit {
				return STOP_ERROR
			}

			return nil
		})
	}

	if err == STOP_ERROR {
		return
	}

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
