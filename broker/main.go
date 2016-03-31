package main

import (
	"database/sql"
	"flag"
	"fmt"
	conf "github.com/FidelityInternational/chaos-galago/broker/config"
	utils "github.com/FidelityInternational/chaos-galago/broker/utils"
	webs "github.com/FidelityInternational/chaos-galago/broker/web_server"
	"net/http"
	"os"
)

// Options struct
type Options struct {
	ConfigPath string
}

var options Options

func init() {
	defaultConfigPath := utils.GetPath([]string{"assets", "config.json"})
	flag.StringVar(&options.ConfigPath, "c", defaultConfigPath, "use '-c' option to specify the config file path")

	flag.Parse()
}

func main() {
	var err error
	_, err = conf.LoadConfig(options.ConfigPath)
	if err != nil {
		panic(fmt.Sprintf("Error loading config file [%s]...", err.Error()))
	}

	server, err := webs.CreateServer(dbConn, webs.CreateController)
	if err != nil {
		panic(fmt.Sprintf("Error creating server [%s]...", err.Error()))
	}

	router := server.Start()

	http.Handle("/", router)

	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		fmt.Println("ListenAndServe:", err)
	}
}

func dbConn(driverName string, connectionString string) (*sql.DB, error) {
	db, err := sql.Open(driverName, connectionString)
	return db, err
}
