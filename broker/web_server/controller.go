package webServer

import (
	"database/sql"
	"fmt"
	sharedModel "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/chaos-galago/shared/model"
	"github.com/FidelityInternational/chaos-galago/broker/config"
	model "github.com/FidelityInternational/chaos-galago/broker/model"
	utils "github.com/FidelityInternational/chaos-galago/broker/utils"
	"net/http"
	"os"
	"strconv"
)

const (
	defaultPollingIntervalSeconds = 10
)

// Controller struct
type Controller struct {
	DB   *sql.DB
	Conf *config.Config
}

// CreateController - returns a populated controller object
func CreateController(db *sql.DB, conf *config.Config) *Controller {
	return &Controller{
		DB:   db,
		Conf: conf,
	}
}

// Catalog - returns the service catalog
func (c *Controller) Catalog(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Service Broker Catalog...")

	var catalog model.Catalog
	var catalogPath string
	catalogFileName := "catalog.json"

	if os.Getenv("CATALOG_PATH") != "" {
		catalogPath = os.Getenv("CATALOG_PATH")
	} else if c.Conf.CatalogPath != "" {
		catalogPath = c.Conf.CatalogPath
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	err := utils.ReadAndUnmarshal(&catalog, catalogPath, catalogFileName)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusOK, catalog)
}

// CreateServiceInstance - creates a service instance
func (c *Controller) CreateServiceInstance(w http.ResponseWriter, r *http.Request) {
	var (
		instance        sharedModel.ServiceInstance
		vcapApplication model.VCAPApplication
		probability     float64
		frequency       int
	)
	fmt.Println("Create Service Instance...")

	err := utils.ProvisionDataFromRequest(r.Body, &instance)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = utils.GetVCAPApplicationVars(&vcapApplication)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	applicationURIUnparsed := vcapApplication.ApplicationURIs[0]
	applicationURI := utils.RemoveGreenFromURI(applicationURIUnparsed)

	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")

	if os.Getenv("PROBABILITY") != "" {
		probability, _ = strconv.ParseFloat(os.Getenv("PROBABILITY"), 64)
	} else if c.Conf.DefaultProbability != 0 {
		probability = c.Conf.DefaultProbability
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	if os.Getenv("FREQUENCY") != "" {
		frequency, _ = strconv.Atoi(os.Getenv("FREQUENCY"))
	} else if c.Conf.DefaultFrequency != 0 {
		frequency = c.Conf.DefaultFrequency
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	instance.DashboardURL = fmt.Sprintf("https://%s/dashboard/%s", applicationURI, instanceID)
	instance.ID = instanceID
	// hard coding as default as it is the only available plan
	instance.PlanID = "default"
	instance.Probability = probability
	instance.Frequency = frequency

	err = utils.AddServiceInstance(c.DB, instance)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := model.CreateServiceInstanceResponse{
		DashboardURL: instance.DashboardURL,
		Probability:  instance.Probability,
		Frequency:    instance.Frequency,
	}
	utils.WriteResponse(w, http.StatusCreated, response)
}

// GetServiceInstance - Returns a service instance
func (c *Controller) GetServiceInstance(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Service Instance State....")

	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance, err := utils.GetServiceInstance(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if instance == (sharedModel.ServiceInstance{}) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response := model.CreateServiceInstanceResponse{
		DashboardURL: instance.DashboardURL,
		Probability:  instance.Probability,
		Frequency:    instance.Frequency,
	}
	utils.WriteResponse(w, http.StatusOK, response)
}

// RemoveServiceInstance - deletes a service instance
func (c *Controller) RemoveServiceInstance(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Remove Service Instance...")

	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance, err := utils.GetServiceInstance(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if instance == (sharedModel.ServiceInstance{}) {
		w.WriteHeader(http.StatusGone)
		return
	}

	err = c.DeleteAssociatedBindings(instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = utils.DeleteServiceInstance(c.DB, instance)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusOK, "{}")
}

// DeleteAssociatedBindings - deletes all binding associated with a service instance
func (c *Controller) DeleteAssociatedBindings(instanceID string) error {
	err := utils.DeleteServiceInstanceBindings(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// Bind - bins a service instance
func (c *Controller) Bind(w http.ResponseWriter, r *http.Request) {
	var binding sharedModel.ServiceBinding
	fmt.Println("Bind Service Instance...")

	bindingID := utils.ExtractVarsFromRequest(r, "service_binding_guid")
	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")

	err := utils.ProvisionDataFromRequest(r.Body, &binding)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	instance, err := utils.GetServiceInstance(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if instance == (sharedModel.ServiceInstance{}) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	probability := instance.Probability
	frequency := instance.Frequency

	credential := model.Credential{
		Probability: probability,
		Frequency:   frequency,
	}

	response := model.CreateServiceBindingResponse{
		Credentials: credential,
	}
	if binding.AppID == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	binding.ID = bindingID
	binding.ServicePlanID = instance.PlanID
	binding.ServiceInstanceID = instance.ID

	err = utils.AddServiceBinding(c.DB, binding)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusCreated, response)
}

// UnBind - unbinds a service instance
func (c *Controller) UnBind(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Unbind Service Instance...")

	bindingID := utils.ExtractVarsFromRequest(r, "service_binding_guid")
	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance, err := utils.GetServiceInstance(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if instance == (sharedModel.ServiceInstance{}) {
		w.WriteHeader(http.StatusGone)
		return
	}

	err = utils.DeleteServiceBinding(c.DB, bindingID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusOK, "{}")
}

// GetDashboard - returns the dashboard
func (c *Controller) GetDashboard(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Loading Dashboard...")

	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance, err := utils.GetServiceInstance(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if instance == (sharedModel.ServiceInstance{}) {
		w.WriteHeader(http.StatusGone)
		return
	}

	response := fmt.Sprintf(`<html>
	<head>
		<link rel="stylesheet" href="/css/bootstrap.min.css">
		<link rel="stylesheet" href="/css/bootstrap-theme.min.css">
		<script src="/js/jquery-1.11.3.min.js"></script>
		<script src="/js/bootstrap.min.js"></script>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<title>Dashboard</title>
	</head>
	<body>
		<div class="container">
			<h1>Change Service Instance Config</h1>
			<form action="/dashboard/%s" method="POST">
				<fieldset class="form-group">
					<label for "probability">Probability</label>
					<input type="number" step="0.01" min="0" max="1" class="form-control" id="probability" name="probability" placeholder="%v">
				</fieldset>
				<fieldset class="form-group">
					<label for "frequency">Frequency</label>
					<input type="number" min="1" max="60" class="form-control" id="frequency" name="frequency" placeholder="%v">
				</fieldset>
				<div class="form-group row">
					<button type="submit" class="btn btn-primary">Submit</button>
				</div>
			</form>
		</div>
	</body>
</html>
`, instanceID, instance.Probability, instance.Frequency)

	utils.WriteResponse(w, http.StatusOK, response)
}

// UpdateServiceInstance - updates a service instance
func (c *Controller) UpdateServiceInstance(w http.ResponseWriter, r *http.Request) {
	var valid = true
	fmt.Println("Updating Service Instance...")

	instanceID := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance, err := utils.GetServiceInstance(c.DB, instanceID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if instance == (sharedModel.ServiceInstance{}) {
		w.WriteHeader(http.StatusGone)
		return
	}

	probability, _ := strconv.ParseFloat(r.FormValue("probability"), 64)
	frequency, _ := strconv.Atoi(r.FormValue("frequency"))

	if !(probability >= 0 && probability <= 1) {
		fmt.Printf("\nProbability: %v\n", probability)
		valid = false
	}

	if !(frequency >= 1 && frequency <= 60) {
		fmt.Printf("\nFrequency: %v\n", frequency)
		valid = false
	}

	if valid {
		err = utils.UpdateServiceInstance(c.DB, instanceID, probability, frequency)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response := fmt.Sprintf(`<html>
	<head>
		<link rel="stylesheet" href="/css/bootstrap.min.css">
		<link rel="stylesheet" href="/css/bootstrap-theme.min.css">
		<script src="/js/jquery-1.11.3.min.js"></script>
		<script src="/js/bootstrap.min.js"></script>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<title>Dashboard</title>
	</head>
	<body>
		<div class="container">
			<h1>New Service Instance Configuration</h1>
			<p>Probability: %v</p>
			<p>Frequency: %v</p>
		</div>
	</body>
</html>`, probability, frequency)
		utils.WriteResponse(w, http.StatusAccepted, response)
	} else {
		response := `<html>
	<head>
		<link rel="stylesheet" href="/css/bootstrap.min.css">
		<link rel="stylesheet" href="/css/bootstrap-theme.min.css">
		<script src="/js/jquery-1.11.3.min.js"></script>
		<script src="/js/bootstrap.min.js"></script>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<title>Dashboard</title>
	</head>
	<body>
		<div class="container">
			<h1>Invalid Configuration Request</h1>
			<p>Probability must be between 0 and 1</p>
			<p>Frequency must be between 1 and 60</p>
		</div>
	</body>
</html>`
		utils.WriteResponse(w, http.StatusBadRequest, response)
	}
}
