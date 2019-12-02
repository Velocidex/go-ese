package main

import (
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"www.velocidex.com/golang/go-ese/parser"
)

type CommandHandler func(command string) bool

var (
	app = kingpin.New("eseparser",
		"A tool for inspecting ese files.")

	debug = app.Flag("debug", "Enable debug messages").Bool()

	command_handlers []CommandHandler
)

func main() {
	app.HelpFlag.Short('h')
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debug {
		parser.Debug = true
		parser.DebugWalk = true
	}

	for _, command_handler := range command_handlers {
		if command_handler(command) {
			break
		}
	}
}
