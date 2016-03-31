package utils

import (
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/chaos-galago/shared/model"
	// sql driver
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type ioRead func(ioReader io.Reader) ([]byte, error)

// RemoveGreenFromURI - removes "-green" from provided URI for zero downtime deployments
func RemoveGreenFromURI(URI string) string {
	return strings.Replace(URI, "-green", "", 1)
}

// GetVCAPApplicationVars - populates an object based on the "VCAP_APPLICATION" environment variables
func GetVCAPApplicationVars(object interface{}) error {
	vcapApplication := os.Getenv("VCAP_APPLICATION")
	err := json.Unmarshal([]byte(vcapApplication), object)
	if err != nil {
		return err
	}
	return nil
}

// ReadAndUnmarshal - loads file into object
func ReadAndUnmarshal(object interface{}, dir string, fileName string) error {
	path := dir + string(os.PathSeparator) + fileName

	bytes, err := ReadFile(path, ioutil.ReadAll)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, object)
	if err != nil {
		return err
	}

	return nil
}

// SetupInstanceDB - creates the service_instances DB if it does not exist
func SetupInstanceDB(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS service_instances
	(
		id varchar(255),
		dashboardURL varchar(255),
		planID varchar(255),
		probability decimal(2,2),
		frequency int
	)`)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SetupBindingDB - creates the service_bindings DB if it does not exist
func SetupBindingDB(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS service_bindings
	(
		id varchar(255),
		appID varchar(255),
		servicePlanID varchar(255),
		serviceInstanceID varchar(255),
		lastProcessed varchar(255)
	)`)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// UpdateServiceInstance - update service_instances database
func UpdateServiceInstance(db *sql.DB, serviceInstanceID string, probability float64, frequency int) error {
	_, err := db.Exec("UPDATE service_instances SET probability=?,frequency=? WHERE id=?", probability, frequency, serviceInstanceID)
	if err != nil {
		return err
	}
	return nil
}

// GetServiceInstance - loads a service instance to memory from database
func GetServiceInstance(db *sql.DB, serviceInstanceID string) (sharedModel.ServiceInstance, error) {
	var (
		id, dashboardURL, planID string
		probability              float64
		frequency                int
	)

	row := db.QueryRow("SELECT * FROM service_instances WHERE id=?", serviceInstanceID)
	err := row.Scan(&id, &dashboardURL, &planID, &probability, &frequency)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return sharedModel.ServiceInstance{}, nil
		}
		return sharedModel.ServiceInstance{}, err
	}
	if id == "" {
		return sharedModel.ServiceInstance{}, errors.New("ID cannot be nil")
	}
	return sharedModel.ServiceInstance{ID: id, DashboardURL: dashboardURL, PlanID: planID, Probability: probability, Frequency: frequency}, nil
}

// DeleteServiceInstanceBindings - deletes from service_bindings based on service instance ID
func DeleteServiceInstanceBindings(db *sql.DB, serviceInstanceID string) error {
	_, err := db.Exec("DELETE FROM service_bindings WHERE serviceInstanceID=?", serviceInstanceID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteServiceBinding - deletes from service_bindings based on service binding ID
func DeleteServiceBinding(db *sql.DB, serviceBindingID string) error {
	_, err := db.Exec("DELETE FROM service_bindings WHERE id=?", serviceBindingID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteServiceInstance - deletes from service_instances based on service instance ID
func DeleteServiceInstance(db *sql.DB, serviceInstance sharedModel.ServiceInstance) error {
	_, err := db.Exec("DELETE FROM service_instances WHERE id=?", serviceInstance.ID)
	if err != nil {
		return err
	}
	return nil
}

// AddServiceBinding - adds a row to service_bindings database
func AddServiceBinding(db *sql.DB, serviceBinding sharedModel.ServiceBinding) error {
	_, err := db.Exec("INSERT INTO service_bindings VALUES (?, ?, ?, ?, ?)", serviceBinding.ID, serviceBinding.AppID, serviceBinding.ServicePlanID, serviceBinding.ServiceInstanceID, serviceBinding.LastProcessed)
	if err != nil {
		return err
	}
	return nil
}

// AddServiceInstance - adds a row to service_isntances database
func AddServiceInstance(db *sql.DB, serviceInstance sharedModel.ServiceInstance) error {
	_, err := db.Exec("INSERT INTO service_instances VALUES (?, ?, ?, ?, ?)", serviceInstance.ID, serviceInstance.DashboardURL, serviceInstance.PlanID, serviceInstance.Probability, serviceInstance.Frequency)
	if err != nil {
		return err
	}
	return nil
}

// WriteResponse - creates an http response
func WriteResponse(w http.ResponseWriter, code int, object interface{}) {
	var (
		data []byte
		err  error
	)

	if str, ok := object.(string); ok {
		data = []byte(str)
	} else {
		data, err = json.Marshal(object)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(code)
	fmt.Fprintf(w, string(data))
}

// ProvisionDataFromRequest - Unmarhsals json to object
func ProvisionDataFromRequest(r io.Reader, object interface{}) error {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, object)
	if err != nil {
		return err
	}

	return nil
}

// ExtractVarsFromRequest - extracts variables from http request
func ExtractVarsFromRequest(r *http.Request, varName string) string {
	return mux.Vars(r)[varName]
}

// ReadFile - loads a file to a byte array
func ReadFile(path string, ioRead ioRead) (content []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	bytes, err := ioRead(file)
	if err != nil {
		return
	}
	content = bytes

	return
}

// GetPath - builds a path string using os native path separators
func GetPath(paths []string) string {
	workDirectory, _ := os.Getwd()

	if len(paths) == 0 {
		return workDirectory
	}

	resultPath := workDirectory

	for _, path := range paths {
		resultPath += string(os.PathSeparator)
		resultPath += path
	}

	return resultPath
}
