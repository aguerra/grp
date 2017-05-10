package main

import (
	"log"

	"github.com/aguerra/grp/server"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.LUTC)
	var conf server.ServerConfig
	if err := envconfig.Process("grp", &conf); err != nil {
		log.Fatal(err)
	}
	srv := server.New(&conf)
	log.Fatal(srv.ListenAndServe())
}
