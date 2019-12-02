package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"www.velocidex.com/golang/go-ese/parser"
)

var (
	page_command = app.Command(
		"page", "Dump information about a database page")

	page_command_file_arg = page_command.Arg(
		"file", "The image file to inspect",
	).Required().OpenFile(os.O_RDONLY, os.FileMode(0666))

	page_command_page_number = page_command.Arg(
		"page_number", "The page to inspect",
	).Required().Int64()
)

func doPage() {
	ese_ctx, err := parser.NewESEContext(*page_command_file_arg)
	kingpin.FatalIfError(err, "Unable to open ese file")

	parser.DumpPage(ese_ctx, *page_command_page_number)
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case page_command.FullCommand():
			doPage()
		default:
			return false
		}
		return true
	})
}
