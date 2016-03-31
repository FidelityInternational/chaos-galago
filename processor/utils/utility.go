package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/chaos-galago/shared/model"
	"github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/chaos-galago/shared/utils"
	"github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/github.com/cloudfoundry-community/go-cfclient"
	"github.com/FidelityInternational/chaos-galago/processor/model"
	"math/rand"
	"os"
	"strconv"
	"time"
)

// ShouldRun - determins of chaos should be run based on probability
func ShouldRun(probability float64) bool {
	return rand.Float64() <= probability
}

// IsAppHealthy - determins if an app is currently in "RUNNING" state
func IsAppHealthy(appInstances map[string]cfclient.AppInstance) bool {
	for _, instance := range appInstances {
		if instance.State != "RUNNING" {
			return false
		}
	}
	return true
}

// PickAppInstance - selects an app instance index
func PickAppInstance(appInstances map[string]cfclient.AppInstance) int {
	for k := range appInstances {
		value, _ := strconv.Atoi(k)
		return value
	}
	return 0
}

// ShouldProcess - determines if chaos-galago should be run based on frequency and previous run
func ShouldProcess(frequency int, lastProcessed string) bool {
	if lastProcessed == "" {
		return true
	}
	layout := "2006-01-02T15:04:05Z"
	timeStamp, err := time.Parse(layout, lastProcessed)
	if err != nil {
		return false
	}
	duration := time.Duration(-frequency) * time.Minute
	if timeStamp.UTC().Before(time.Now().UTC().Add(duration)) {
		return true
	}
	return false
}

// GetBoundApps - Loads bound apps into memory from database
func GetBoundApps(db *sql.DB) []model.Service {
	var services []model.Service
	serviceInstances, err := sharedUtils.ReadServiceInstances(db)
	if err != nil {
		return services
	}
	serviceBindings, err := sharedUtils.ReadServiceBindings(db)
	if err != nil {
		return services
	}
OUTER:
	for _, binding := range serviceBindings {
		serviceInstance := serviceInstances[binding.ServiceInstanceID]
		appID := binding.AppID
		probability := serviceInstance.Probability
		frequency := serviceInstance.Frequency
		if serviceInstance == (sharedModel.ServiceInstance{}) || appID == "" || probability == 0 || frequency == 0 {
			continue OUTER
		}
		services = append(services, model.Service{AppID: appID, LastProcessed: binding.LastProcessed, Probability: probability, Frequency: frequency})
	}
	return services
}

// TimeNow - Formatted current time
func TimeNow() string {
	layout := "2006-01-02T15:04:05Z"
	return time.Now().UTC().Format(layout)
}

// UpdateLastProcessed - writes the last processed time to service_bindings database
func UpdateLastProcessed(db *sql.DB, appID string, lastProcessed string) error {
	_, err := db.Exec("UPDATE service_bindings SET lastProcessed=? WHERE appID=?", lastProcessed, appID)
	if err != nil {
		return err
	}
	return nil
}

// LoadCFConfig - Loads "cf-service" UPS config
func LoadCFConfig() *cfclient.Config {
	var (
		cfServices model.CFServices
	)

	vcapServicesEnv := os.Getenv("VCAP_SERVICES")
	err := json.Unmarshal([]byte(vcapServicesEnv), &cfServices)
	if err != nil {
		return &cfclient.Config{}
	}

	for _, cfService := range cfServices.CFService {
		if cfService.Name == "cf-service" {
			return &cfclient.Config{
				ApiAddress:        fmt.Sprintf("https://api.%s", cfService.Credentials.Domain),
				LoginAddress:      fmt.Sprintf("https://login.%s", cfService.Credentials.Domain),
				Username:          cfService.Credentials.Username,
				Password:          cfService.Credentials.Password,
				SkipSslValidation: cfService.Credentials.SkipSslValidation,
			}
		}
	}

	return &cfclient.Config{}
}
