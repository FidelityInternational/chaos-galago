package sharedUtils

import (
	"database/sql"
	"fmt"
	"github.com/FidelityInternational/chaos-galago/shared/model"
	"github.com/cloudfoundry-community/go-cfenv"

	// sql Driver
	_ "github.com/go-sql-driver/mysql"
)

// ReadServiceInstances - Loads service instances to memory from Database
func ReadServiceInstances(db *sql.DB) (map[string]sharedModel.ServiceInstance, error) {
	var (
		rows                *sql.Rows
		err                 error
		serviceInstancesMap map[string]sharedModel.ServiceInstance
	)
	serviceInstancesMap = make(map[string]sharedModel.ServiceInstance)
	rows, err = db.Query("SELECT * FROM service_instances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, dashboardURL, planID string
			probability              float64
			frequency                int
		)
		if err = rows.Scan(&id, &dashboardURL, &planID, &probability, &frequency); err != nil {
			return nil, err
		}
		serviceInstance := sharedModel.ServiceInstance{ID: id, DashboardURL: dashboardURL, PlanID: planID, Probability: probability, Frequency: frequency}
		serviceInstancesMap[id] = serviceInstance
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return serviceInstancesMap, nil
}

// ReadServiceBindings - Loads service bindings to memory from Database
func ReadServiceBindings(db *sql.DB) (map[string]sharedModel.ServiceBinding, error) {
	var (
		rows               *sql.Rows
		err                error
		serviceBindingsMap map[string]sharedModel.ServiceBinding
	)

	serviceBindingsMap = make(map[string]sharedModel.ServiceBinding)
	rows, err = db.Query("SELECT * FROM service_bindings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, appID, servicePlanID, serviceInstanceID, lastProcessed string
		if err = rows.Scan(&id, &appID, &servicePlanID, &serviceInstanceID, &lastProcessed); err != nil {
			return nil, err
		}
		serviceBinding := sharedModel.ServiceBinding{ID: id, AppID: appID, ServicePlanID: servicePlanID, ServiceInstanceID: serviceInstanceID, LastProcessed: lastProcessed}
		serviceBindingsMap[id] = serviceBinding
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return serviceBindingsMap, nil
}

// GetDBConnectionDetails - Loads database connection details from UPS "chaos-galago-db"
func GetDBConnectionDetails() (string, error) {
	appEnv, err := cfenv.Current()
	if err != nil {
		return "", err
	}

	service, err := appEnv.Services.WithName("chaos-galago-db")
	if err != nil {
		return "", err
	}

	hostname := service.Credentials["host"]
	if nil == hostname {
		hostname = service.Credentials["hostname"]
	}

	database := service.Credentials["database"]
	if nil == database {
		database = service.Credentials["name"]
	}

	dbConnString := fmt.Sprintf("%s:%s@tcp(%s:%v)/%s",
		service.Credentials["username"], service.Credentials["password"], hostname,
		service.Credentials["port"], database)

	return dbConnString, nil
}
