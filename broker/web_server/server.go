package webServer

import (
	"fmt"
	"net/http"

	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/gorilla/mux"

	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/chaos-galago/shared/utils"
	// mysql driver
	"database/sql"
	_ "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/FidelityInternational/chaos-galago/broker/config"
	"github.com/FidelityInternational/chaos-galago/broker/utils"
)

var (
	db                 *sql.DB
	err                error
	dbConnectionString string
)

// Server struct
type Server struct {
	Controller *Controller
}

// DBConn - database opening interface
type DBConn func(driverName string, connectionString string) (*sql.DB, error)

// ControllerCreator - controller creation function
type ControllerCreator func(db *sql.DB, conf *config.Config) *Controller

// CreateServer - creates a server
func CreateServer(dbConnFunc DBConn, controllerCreator ControllerCreator) (*Server, error) {
	dbConnectionString, err = sharedUtils.GetDBConnectionDetails()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	db, err = dbConnFunc("mysql", dbConnectionString)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = utils.SetupInstanceDB(db)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	err = utils.SetupBindingDB(db)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	conf := config.GetConfig()
	controller := controllerCreator(db, conf)

	return &Server{
		Controller: controller,
	}, nil
}

// Start - starts the web server
func (s *Server) Start() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/v2/catalog", s.Controller.Catalog).Methods("GET")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", s.Controller.GetServiceInstance).Methods("GET")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", s.Controller.CreateServiceInstance).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}", s.Controller.RemoveServiceInstance).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", s.Controller.Bind).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{service_instance_guid}/service_bindings/{service_binding_guid}", s.Controller.UnBind).Methods("DELETE")
	router.HandleFunc("/dashboard/{service_instance_guid}", s.Controller.GetDashboard).Methods("GET")
	router.HandleFunc("/dashboard/{service_instance_guid}", s.Controller.UpdateServiceInstance).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web_server/resources/")))

	return router
}
