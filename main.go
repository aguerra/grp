package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aguerra/grp/server"
	log "github.com/inconshreveable/log15"
	logext "github.com/inconshreveable/log15/ext"
	"github.com/kelseyhightower/envconfig"
)

var version string
var showVersion bool

func init() {
	h := logext.FatalHandler(log.CallerFileHandler(log.StdoutHandler))
	log.Root().SetHandler(h)
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()
}

func main() {
	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}
	var conf server.ServerConfig
	if err := envconfig.Process("grp", &conf); err != nil {
		log.Crit("failed to load env", "err", err)
	}
	srv := server.New(&conf)
	log.Crit("grp exited", "err", srv.ListenAndServe())
}
