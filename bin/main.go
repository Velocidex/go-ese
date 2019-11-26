package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type CommandHandler func(command string) bool

var (
	app = kingpin.New("eseparser",
		"A tool for inspecting ese files.")
	command_handlers []CommandHandler
)

func main() {
	app.HelpFlag.Short('h')
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	for _, command_handler := range command_handlers {
		if command_handler(command) {
			break
		}
	}
}
