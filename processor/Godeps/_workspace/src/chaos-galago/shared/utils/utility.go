package sharedUtils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/chaos-galago/shared/model"
	_ "github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"os"
)

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

func GetDBConnectionDetails() (string, error) {
	var (
		vcapServices sharedModel.VCAPServices
		dbConnString string
	)

	vcapServicesEnv := os.Getenv("VCAP_SERVICES")
	err := json.Unmarshal([]byte(vcapServicesEnv), &vcapServices)
	if err != nil {
		return "", err
	}

	for _, ups := range vcapServices.UserProvided {
		if ups.Name == "chaos-galago-db" {
			dbConnString = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", ups.Credentials.Username, ups.Credentials.Password, ups.Credentials.Host, ups.Credentials.Port, ups.Credentials.Database)
			break
		}
	}

	return dbConnString, nil
}
