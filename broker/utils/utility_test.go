package utils_test

import (
	"bytes"
	"fmt"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/chaos-galago/shared/model"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/DATA-DOG/go-sqlmock"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/gorilla/mux"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/FidelityInternational/chaos-galago/broker/model"
	"github.com/FidelityInternational/chaos-galago/broker/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
)

type stubReader struct{}

func (reader stubReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("An error has occured: %s", "ioread error")
}

func mockioReader(ioReader io.Reader) ([]byte, error) {
	return []byte{}, fmt.Errorf("An error has occured: %s", "IO Reader could not be read")
}

func Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/extract_vars_test/{example_var}", ExtractVarsFromRequestTest)
	return r
}

func ExtractVarsFromRequestTest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprintf(w, utils.ExtractVarsFromRequest(r, "example_var"))
}

func init() {
	http.Handle("/", Router())
}

var _ = Describe("#ExtractVarsFromRequest", func() {
	It("Returns a string of the extracted var", func() {
		mockRecorder := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "http://example.com/extract_vars_test/example_value", nil)
		Router().ServeHTTP(mockRecorder, req)
		Expect(mockRecorder.Body.String()).To(Equal("example_value"))
	})
})

var _ = Describe("#ProvisionDataFromRequest", func() {
	var (
		req *http.Request
	)

	Context("When the request body can be read", func() {
		var requestBody io.Reader

		JustBeforeEach(func() {
			req, _ = http.NewRequest("POST", "http://example.com", requestBody)
		})

		Context("and the body can be unmarshalled into the provided object", func() {
			var exampleObject model.Service
			BeforeEach(func() {
				requestBody = bytes.NewReader([]byte(`{"name":"provision_data_test"}`))
			})

			It("populates the passed object", func() {
				err := utils.ProvisionDataFromRequest(req.Body, &exampleObject)
				Expect(err).To(BeNil())
				Expect(exampleObject.Name).To(Equal("provision_data_test"))
			})
		})

		Context("and the body cannot be unmarshalled into the provided object", func() {
			It("Raises an error", func() {
				Expect(utils.ProvisionDataFromRequest(req.Body, "test").Error()).To(Equal("unexpected end of JSON input"))
			})
		})
	})

	Context("when body cannot be read", func() {
		It("Raises an error", func() {
			Expect(utils.ProvisionDataFromRequest(stubReader{}, "test").Error()).To(Equal("An error has occured: ioread error"))
		})
	})
})

var _ = Describe("#WriteResponse", func() {
	Context("When the object can be marhaled into JSON", func() {
		It("Raises an error 500", func() {
			mockRecorder := httptest.NewRecorder()
			exampleObject := make(map[int]string)
			utils.WriteResponse(mockRecorder, 200, exampleObject)
			Expect(mockRecorder.Code).To(Equal(500))
		})
	})

	Context("When the object cannot be marhaled into JSON", func() {
		It("Returns a 200 response with body", func() {
			mockRecorder := httptest.NewRecorder()
			exampleObject := model.CreateServiceInstanceResponse{DashboardURL: "example.com", Probability: 0.2, Frequency: 5}
			utils.WriteResponse(mockRecorder, 200, exampleObject)
			Expect(mockRecorder.Code).To(Equal(200))
			Expect(mockRecorder.Body.String()).To(Equal(`{"dashboard_url":"example.com","probability":0.2,"frequency":5}`))
		})
	})

	Context("When the object is a string", func() {
		It("returns the string value", func() {
			mockRecorder := httptest.NewRecorder()
			exampleObject := "Just a nice string"
			utils.WriteResponse(mockRecorder, 200, exampleObject)
			Expect(mockRecorder.Code).To(Equal(200))
			Expect(mockRecorder.Body.String()).To(Equal(`Just a nice string`))
		})
	})
})

var _ = Describe("#ReadAndUnmarshal", func() {
	var exampleObject model.Catalog
	Context("When the file does not exist", func() {
		It("Raises an error", func() {
			Expect(utils.ReadAndUnmarshal(&exampleObject, "invalid_path", "invalid_file.json")).ToNot(BeNil())
		})
	})

	Context("When the file does exist", func() {
		Context("and json is invalid", func() {
			It("raises an error", func() {
				Expect(utils.ReadAndUnmarshal(&exampleObject, "fixtures", "invalid_file.json")).ToNot(BeNil())
			})
		})

		Context("and the json is valid", func() {
			Context("and the json does not match the schema of exampleObject", func() {
				It("leaves exampleOnject as empty", func() {
					Expect(utils.ReadAndUnmarshal(&exampleObject, "fixtures", "invalid_schema.json")).To(BeNil())
					Expect(exampleObject.Services).To(HaveLen(0))
				})
			})

			Context("and the json does match the schema of exampleObject", func() {
				It("populates exampleObject", func() {
					Expect(utils.ReadAndUnmarshal(&exampleObject, "fixtures", "example_catalog.json")).To(BeNil())
					Expect(exampleObject.Services).To(HaveLen(1))
					Expect(exampleObject.Services[0].Name).To(Equal("chaos-galago-test"))
				})
			})
		})
	})
})

var _ = Describe("#UpdateServiceInstance", func() {
	It("Updates the service instance", func() {
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()

		probability := 0.2
		frequency := 5
		instanceID := "test"

		mock.ExpectExec("UPDATE service_instances.*").WithArgs(probability, frequency, instanceID).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.UpdateServiceInstance(db, instanceID, probability, frequency)).To(BeNil())
	})

	Context("When the sql update command raies an error", func() {
		It("returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			probability := 0.2
			frequency := 5
			instanceID := "test"

			mock.ExpectExec("UPDATE service_instances.*").WithArgs(probability, frequency, instanceID).WillReturnError(fmt.Errorf("An error has occured: %s", "UPDATE error"))
			err = utils.UpdateServiceInstance(db, instanceID, probability, frequency)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: UPDATE error"))
		})
	})
})

var _ = Describe("#DeleteServiceInstance", func() {
	It("Deletes service instance from the database", func() {
		var instance sharedModel.ServiceInstance
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()

		probability := 0.2
		frequency := 5
		planID := "default"
		instanceID := "test"
		applicationURI := "example.com"
		dashboardURL := fmt.Sprintf("https://%s/dashboard/%s", applicationURI, instanceID)

		instance.DashboardURL = dashboardURL
		instance.ID = instanceID
		instance.PlanID = planID
		instance.Probability = probability
		instance.Frequency = frequency

		mock.ExpectExec("DELETE FROM service_instances WHERE id=").WithArgs(instanceID).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.DeleteServiceInstance(db, instance)).To(BeNil())
	})

	Context("When the sql delete command raises and error", func() {
		It("returns an error", func() {
			var instance sharedModel.ServiceInstance
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			probability := 0.2
			frequency := 5
			planID := "default"
			instanceID := "test"
			applicationURI := "example.com"
			dashboardURL := fmt.Sprintf("https://%s/dashboard/%s", applicationURI, instanceID)

			instance.DashboardURL = dashboardURL
			instance.ID = instanceID
			instance.PlanID = planID
			instance.Probability = probability
			instance.Frequency = frequency

			mock.ExpectExec("DELETE FROM service_instances WHERE id=").WithArgs(instanceID).WillReturnError(fmt.Errorf("An error has occured: %s", "DELETE error"))
			err = utils.DeleteServiceInstance(db, instance)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: DELETE error"))
		})
	})
})

var _ = Describe("#DeleteServiceBinding", func() {
	It("Deletes service instance from the database", func() {
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()

		bindingID := "test"

		mock.ExpectExec("DELETE FROM service_bindings WHERE id=").WithArgs(bindingID).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.DeleteServiceBinding(db, bindingID)).To(BeNil())
	})

	Context("When the sql delete command raises and error", func() {
		It("returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			bindingID := "test"

			mock.ExpectExec("DELETE FROM service_bindings WHERE id=").WithArgs(bindingID).WillReturnError(fmt.Errorf("An error has occured: %s", "DELETE error"))
			err = utils.DeleteServiceBinding(db, bindingID)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: DELETE error"))
		})
	})
})

var _ = Describe("#DeleteServiceInstanceBindings", func() {
	It("Deletes service instance bindings from database", func() {
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()

		instanceID := "test"

		mock.ExpectExec("DELETE FROM service_bindings WHERE serviceInstanceID=").WithArgs(instanceID).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.DeleteServiceInstanceBindings(db, instanceID)).To(BeNil())
	})

	Context("When the sql delete command raises and error", func() {
		It("returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			instanceID := "test"

			mock.ExpectExec("DELETE FROM service_bindings WHERE serviceInstanceID=").WithArgs(instanceID).WillReturnError(fmt.Errorf("An error has occured: %s", "DELETE error"))
			err = utils.DeleteServiceInstanceBindings(db, instanceID)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: DELETE error"))
		})
	})
})

var _ = Describe("#AddServiceInstance", func() {
	It("Adds service instance details to the database", func() {
		var instance sharedModel.ServiceInstance
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()

		probability := 0.2
		frequency := 5
		planID := "default"
		instanceID := "test"
		applicationURI := "example.com"
		dashboardURL := fmt.Sprintf("https://%s/dashboard/%s", applicationURI, instanceID)

		instance.DashboardURL = dashboardURL
		instance.ID = instanceID
		instance.PlanID = planID
		instance.Probability = probability
		instance.Frequency = frequency

		mock.ExpectExec("INSERT INTO service_instances").WithArgs(instanceID, dashboardURL, planID, probability, frequency).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.AddServiceInstance(db, instance)).To(BeNil())
	})

	Context("When the sql insert command raises and error", func() {
		It("returns an error", func() {
			var instance sharedModel.ServiceInstance
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			probability := 0.2
			frequency := 5
			planID := "default"
			instanceID := "test"
			applicationURI := "example.com"
			dashboardURL := fmt.Sprintf("https://%s/dashboard/%s", applicationURI, instanceID)

			instance.DashboardURL = dashboardURL
			instance.ID = instanceID
			instance.PlanID = planID
			instance.Probability = probability
			instance.Frequency = frequency

			mock.ExpectExec("INSERT INTO service_instances").WithArgs(instanceID, dashboardURL, planID, probability, frequency).WillReturnError(fmt.Errorf("An error has occured: %s", "INSERT error"))
			err = utils.AddServiceInstance(db, instance)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: INSERT error"))
		})
	})
})

var _ = Describe("#AddServiceBinding", func() {
	It("Adds service binding details to the database", func() {
		var binding sharedModel.ServiceBinding
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()
		planID := "default"
		instanceID := "test"
		bindingID := "test"
		appID := "test"

		binding.ID = bindingID
		binding.ServicePlanID = planID
		binding.ServiceInstanceID = instanceID
		binding.AppID = appID

		mock.ExpectExec("INSERT INTO service_bindings").WithArgs(bindingID, appID, planID, instanceID, "").WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.AddServiceBinding(db, binding)).To(BeNil())
	})

	Context("When the sql insert command raises and error", func() {
		It("returns an error", func() {
			var binding sharedModel.ServiceBinding
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()
			planID := "default"
			instanceID := "test"
			bindingID := "test"
			appID := "test"

			binding.ID = bindingID
			binding.ServicePlanID = planID
			binding.ServiceInstanceID = instanceID
			binding.AppID = appID

			mock.ExpectExec("INSERT INTO service_bindings").WithArgs(bindingID, appID, planID, instanceID, "").WillReturnError(fmt.Errorf("An error has occured: %s", "INSERT error"))
			err = utils.AddServiceBinding(db, binding)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: INSERT error"))
		})
	})
})

var _ = Describe("#SetupInstanceDB", func() {
	Context("When the table exists", func() {
		It("does nothing", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(1, 1))
			Expect(utils.SetupInstanceDB(db)).To(BeNil())
		})
	})

	Context("When the table does not exist", func() {
		It("creates the table", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(1, 1))
			Expect(utils.SetupInstanceDB(db)).To(BeNil())
		})
	})

	Context("When the create command returns an error", func() {
		It("Returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectExec("CREATE TABLE").WillReturnError(fmt.Errorf("An error has occured: %s", "Database Create Error"))
			err = utils.SetupInstanceDB(db)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: Database Create Error"))
		})
	})
})

var _ = Describe("#SetupBindingDB", func() {
	Context("When the table exists", func() {
		It("does nothing", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(1, 1))
			Expect(utils.SetupBindingDB(db)).To(BeNil())
		})
	})

	Context("When the table does not exist", func() {
		It("creates the table", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(1, 1))
			Expect(utils.SetupBindingDB(db)).To(BeNil())
		})
	})

	Context("When the create command returns an error", func() {
		It("Returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectExec("CREATE TABLE").WillReturnError(fmt.Errorf("An error has occured: %s", "Database Create Error"))
			err = utils.SetupBindingDB(db)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occured: Database Create Error"))
		})
	})
})

var _ = Describe("#GetServiceInstance", func() {
	var (
		serviceInstance sharedModel.ServiceInstance
	)

	Context("When the service instance exists", func() {
		It("Returns the service instance", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
				AddRow("1", "example.com/1", "1", 0.2, 5)

			mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WillReturnRows(rows)

			serviceInstance, err = utils.GetServiceInstance(db, "1")
			Expect(err).To(BeNil())
			Expect(serviceInstance).To(Equal(sharedModel.ServiceInstance{ID: "1", DashboardURL: "example.com/1", PlanID: "1", Probability: 0.2, Frequency: 5}))
		})
	})

	Context("When the service instance does not exist", func() {
		It("Returns an empty struct", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
			mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WillReturnRows(rows)

			serviceInstance, err = utils.GetServiceInstance(db, "1")
			Expect(err).To(BeNil())
			Expect(serviceInstance).To(Equal(sharedModel.ServiceInstance{}))
		})
	})

	Context("when a row cannot be scanned", func() {
		It("returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability"}).
				AddRow("1", "example.com/1", "1", 0.2)

			mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WillReturnRows(rows)

			serviceInstance, err = utils.GetServiceInstance(db, "1")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("sql: expected 4 destination arguments in Scan, not 5"))
		})
	})

	Context("when id is blank", func() {
		It("returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
				AddRow("", "example.com/1", "1", 0.2, 5)

			mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WillReturnRows(rows)

			serviceInstance, err = utils.GetServiceInstance(db, "1")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("ID cannot be nil"))
		})
	})
})

var _ = Describe("#GetVCAPApplicationVars", func() {
	var (
		object              model.VCAPApplication
		vcapApplicationJSON string
	)

	Context("when VCAP_APPLICATION exists", func() {
		BeforeEach(func() {
			vcapApplicationJSON = `{"application_name": "test", "application_uris": ["test.example.com","test2.example.com"]}`
			os.Setenv("VCAP_APPLICATION", vcapApplicationJSON)
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_APPLICATION")
		})

		Context("and VCAP_APPLICATION can be unmarshalled into the provided object", func() {
			It("populates the object", func() {
				Expect(utils.GetVCAPApplicationVars(&object)).To(BeNil())
				Expect(object.ApplicationName).To(Equal("test"))
				Expect(object.ApplicationURIs[0]).To(Equal("test.example.com"))
				Expect(object.ApplicationURIs[1]).To(Equal("test2.example.com"))
			})
		})
	})

	Context("when VCAP_APPLICATION does not exist", func() {
		It("returns an error", func() {
			Expect(utils.GetVCAPApplicationVars(&object).Error()).To(Equal("unexpected end of JSON input"))
		})
	})
})

var _ = Describe("#GetPath", func() {
	It("Returns the pull path to a file", func() {
		fullPath := utils.GetPath([]string{"test", "test.json"})
		currentDir, _ := os.Getwd()
		Expect(fullPath).To(MatchRegexp("test.json"))
		Expect(fullPath).To(ContainSubstring(currentDir))
	})

	Context("When path is nil", func() {
		It("returns the current working directory", func() {
			fullPath := utils.GetPath([]string{})
			currentDir, _ := os.Getwd()
			Expect(fullPath).To(Equal(currentDir))
		})
	})
})

var _ = Describe("#ReadFile", func() {
	var fullPath string

	Context("When the file exists", func() {
		BeforeEach(func() {
			fullPath = utils.GetPath([]string{"fixtures", "example_catalog.json"})
		})

		Context("and the file can be read", func() {
			It("returns a byte array of the file contents", func() {
				output, err := utils.ReadFile(fullPath, ioutil.ReadAll)
				Expect(err).To(BeNil())
				Expect(output).To(BeAssignableToTypeOf([]byte{}))
				Expect(string(output)).To(MatchRegexp("Provides the ability to cause chaos on bound application instances"))
			})
		})

		Context("and the file cannot be read", func() {
			It("returns an error", func() {
				_, err := utils.ReadFile(fullPath, mockioReader)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("An error has occured: IO Reader could not be read"))
			})
		})
	})

	Context("When the file does not exist", func() {
		BeforeEach(func() {
			fullPath = utils.GetPath([]string{"fixtures", "no_file"})
		})

		It("returns an error", func() {
			_, err := utils.ReadFile(fullPath, ioutil.ReadAll)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(MatchRegexp("open"))
		})
	})
})

var _ = Describe("#RemoveGreenFromURI", func() {
	Context("When the URI contains -green", func() {
		It("Strips -green from the URI", func() {
			Expect(utils.RemoveGreenFromURI("https://test-green.example.com")).To(Equal("https://test.example.com"))
		})
	})

	Context("When the URI does not contain -green", func() {
		It("does nothing", func() {
			Expect(utils.RemoveGreenFromURI("https://test.example.com")).To(Equal("https://test.example.com"))
		})
	})
})
