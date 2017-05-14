package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aguerra/grp/server"
	"github.com/kelseyhightower/envconfig"
)

var version string
var showVersion bool

func init() {
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()
}

func main() {
	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.LUTC)
	var conf server.ServerConfig
	if err := envconfig.Process("grp", &conf); err != nil {
		log.Fatal(err)
	}
	srv := server.New(&conf)
	log.Fatal(srv.ListenAndServe())
}
