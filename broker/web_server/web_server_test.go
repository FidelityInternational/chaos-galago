package webServer_test

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/chaos-galago/shared/model"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/DATA-DOG/go-sqlmock"
	"github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/gorilla/mux"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/FidelityInternational/chaos-galago/broker/config"
	webs "github.com/FidelityInternational/chaos-galago/broker/web_server"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

func Router(controller *webs.Controller) *mux.Router {
	server := &webs.Server{Controller: controller}
	r := server.Start()
	return r
}

func init() {
	var controller *webs.Controller
	http.Handle("/", Router(controller))
}

func mockFailedInstanceDBConn(driverName string, connectionString string) (*sql.DB, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
		os.Exit(1)
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS service_instances.*").WillReturnError(fmt.Errorf("An error has occured: %s", "Database Create Error"))
	return db, err
}

func mockFailedBindingDBConn(driverName string, connectionString string) (*sql.DB, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
		os.Exit(1)
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS service_instances.*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS service_bindings.").WillReturnError(fmt.Errorf("An error has occured: %s", "Database Create Error"))
	return db, err
}

func mockDBConn(driverName string, connectionString string) (*sql.DB, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
		os.Exit(1)
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS service_instances.*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS service_bindings.*").WillReturnResult(sqlmock.NewResult(1, 1))
	return db, err
}

func mockErrDBConn(driverName string, connectionString string) (*sql.DB, error) {
	db, _, err := sqlmock.New()
	err = fmt.Errorf("An error has occured: %s", "Conn String Fetch Error")
	return db, err
}

func mockCreateController(db *sql.DB, conf *config.Config) *webs.Controller {
	return &webs.Controller{
		DB:   &sql.DB{},
		Conf: &config.Config{},
	}
}

var _ = Describe("Server", func() {
	Describe("#CreateServer", func() {
		var vcapServicesJSON string
		BeforeEach(func() {
			vcapServicesJSON = `{
  "user-provided": [
   {
    "credentials": {
    	"username":"test_user",
    	"password":"test_password",
    	"host":"test_host",
    	"port":"test_port",
    	"database":"test_database"
    },
    "label": "user-provided",
    "name": "chaos-galago-db",
    "syslog_drain_url": "",
    "tags": []
   }
  ]
 }`
		})

		JustBeforeEach(func() {
			os.Setenv("VCAP_SERVICES", vcapServicesJSON)
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_SERVICES")
		})

		Context("When chaos-galago-db service is set", func() {
			Context("and fetching the connection string raises an error", func() {
				It("returns an error", func() {
					_, err := webs.CreateServer(mockErrDBConn, mockCreateController)
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(MatchRegexp("An error has occured: Conn String Fetch Error"))
				})
			})

			Context("and fetching the connection string does not raise an error", func() {
				Context("and SetupInstanceDB does not raise an error", func() {
					Context("and SetupBindingDB does not raise an error", func() {
						It("creates a Server object", func() {
							server, err := webs.CreateServer(mockDBConn, mockCreateController)
							Expect(err).To(BeNil())
							Expect(server).To(BeAssignableToTypeOf(&webs.Server{}))
						})
					})
					Context("and SetupBindingDB raises an error", func() {
						It("returns an error", func() {
							_, err := webs.CreateServer(mockFailedBindingDBConn, mockCreateController)
							Expect(err).ToNot(BeNil())
							Expect(err.Error()).To(MatchRegexp("An error has occured: Database Create Error"))
						})
					})
				})

				Context("and SetupInstanceDB raises an error", func() {
					It("returns an error", func() {
						_, err := webs.CreateServer(mockFailedInstanceDBConn, mockCreateController)
						Expect(err).ToNot(BeNil())
						Expect(err.Error()).To(MatchRegexp("An error has occured: Database Create Error"))
					})
				})
			})

			Context("When chaos-galago-db service is not set", func() {
				BeforeEach(func() {
					vcapServicesJSON = `{
  "user-provided": [
   {
    "credenti
    	"password":"test_password",
    	"host":"test_host",
    	"port":"test_port",
    	"database":"test_database"
    },
    "label": "user-provided",
    "name": "chaos-galago-db",
    "syslog_drain_url": "",
    "tags": []
   }
  ]
 }`
				})

				It("returns an error", func() {
					_, err := webs.CreateServer(mockDBConn, mockCreateController)
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(MatchRegexp("invalid"))
				})
			})
		})
	})
})

var _ = Describe("Contoller", func() {
	var (
		db   *sql.DB
		conf *config.Config
		mock sqlmock.Sqlmock
		err  error
	)

	BeforeEach(func() {
		db, mock, err = sqlmock.New()
		conf = &config.Config{}
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
	})

	AfterEach(func() {
		defer db.Close()
	})

	Describe("#CreateController", func() {

		BeforeEach(func() {
		})

		It("Creates a controller", func() {
			controller := webs.CreateController(db, conf)
			Expect(controller).To(BeAssignableToTypeOf(&webs.Controller{}))
		})
	})

	Describe("#DeleteAssociatedBindings", func() {
		var (
			controller *webs.Controller
		)

		BeforeEach(func() {
			controller = webs.CreateController(db, conf)
		})

		Context("When the binding exists", func() {
			It("returns nil", func() {
				mock.ExpectExec("DELETE FROM service_bindings WHERE serviceInstanceID=").WithArgs("1").WillReturnResult(sqlmock.NewResult(1, 1))
				Expect(controller.DeleteAssociatedBindings("1")).To(BeNil())
			})
		})

		Context("When the binding does not exist", func() {
			It("returns nil", func() {
				mock.ExpectExec("DELETE FROM service_bindings WHERE serviceInstanceID=").WithArgs("does_not_exist").WillReturnResult(sqlmock.NewResult(1, 0))
				Expect(controller.DeleteAssociatedBindings("does_not_exist")).To(BeNil())
			})
		})
	})

	Describe("#RemoveServiceInstance", func() {
		var (
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			controller = webs.CreateController(db, conf)
			mockRecorder = httptest.NewRecorder()
		})

		Context("When the service instance exists", func() {
			JustBeforeEach(func() {
				req, _ = http.NewRequest("DELETE", "http://example.com/v2/service_instances/1", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			Context("and the service instance can be received from the DB", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
						AddRow("1", "https://example.com/dashboard/1", "1", 0.2, 5)
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})

				Context("and the service bindings can be deleted", func() {
					BeforeEach(func() {
						mock.ExpectExec("DELETE FROM service_bindings WHERE serviceInstanceID=").WithArgs("1").WillReturnResult(sqlmock.NewResult(1, 1))
					})

					Context("and the service instance can be deleted", func() {
						BeforeEach(func() {
							mock.ExpectExec("DELETE FROM service_instances WHERE id=").WithArgs("1").WillReturnResult(sqlmock.NewResult(1, 1))
						})

						It("Returns a 200", func() {
							Expect(mockRecorder.Code).To(Equal(200))
							Expect(mockRecorder.Body.String()).To(Equal("{}"))
						})
					})

					Context("and the service instance cannot be deleted", func() {
						BeforeEach(func() {
							mock.ExpectExec("DELETE FROM service_instances WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occured: %s", "DB error"))
						})

						It("Returns an error 500", func() {
							Expect(mockRecorder.Code).To(Equal(500))
						})
					})
				})

				Context("and the service bindings cannot be deleted", func() {
					BeforeEach(func() {
						mock.ExpectExec("DELETE FROM service_bindings WHERE serviceInstanceID=").WillReturnError(fmt.Errorf("An error has occured: %s", "DB error"))
					})

					It("Returns an error 500", func() {
						Expect(mockRecorder.Code).To(Equal(500))
					})
				})
			})

			Context("and the service instance cannot be receuve from the DB", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occured: %s", "DB error"))
				})

				It("Returns an error 500", func() {
					Expect(mockRecorder.Code).To(Equal(500))
				})
			})
		})

		Context("When the service instance does not exist", func() {
			BeforeEach(func() {
				rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
				mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("2").WillReturnRows(rows)
				req, _ = http.NewRequest("DELETE", "http://example.com/v2/service_instances/2", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			It("Returns a 410", func() {
				Expect(mockRecorder.Code).To(Equal(410))
			})
		})
	})

	Describe("#Unbind", func() {
		var (
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			controller = webs.CreateController(db, conf)
			mockRecorder = httptest.NewRecorder()
		})

		Context("When the service instance exists", func() {
			JustBeforeEach(func() {
				req, _ = http.NewRequest("DELETE", "http://example.com/v2/service_instances/1/service_bindings/1", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			Context("and the service instance can be fetched", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
						AddRow("1", "https://example.com/dashboard/1", "1", 0.2, 5)
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})

				Context("and the service binding can be deleted", func() {
					BeforeEach(func() {
						mock.ExpectExec("DELETE FROM service_bindings WHERE id=").WithArgs("1").WillReturnResult(sqlmock.NewResult(1, 1))
					})

					It("Returns a 200", func() {
						Expect(mockRecorder.Code).To(Equal(200))
						Expect(mockRecorder.Body.String()).To(Equal("{}"))
					})
				})

				Context("and the service binding cannot be deleted", func() {
					BeforeEach(func() {
						mock.ExpectExec("DELETE FROM service_bindings WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
					})

					It("returns an error 500", func() {
						Expect(mockRecorder.Code).To(Equal(500))
					})
				})
			})

			Context("and the service instance cannot be fetched", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
				})

				It("returns an error 500", func() {
					Expect(mockRecorder.Code).To(Equal(500))
				})
			})
		})

		Context("When the service instance does not exist", func() {
			BeforeEach(func() {
				rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
				mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("2").WillReturnRows(rows)
				req, _ = http.NewRequest("DELETE", "http://example.com/v2/service_instances/2/service_bindings/2", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			It("Returns a 410", func() {
				Expect(mockRecorder.Code).To(Equal(410))
			})
		})
	})

	Describe("#GetDashboard", func() {
		var (
			response     string
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			controller = webs.CreateController(db, conf)
			mockRecorder = httptest.NewRecorder()
			response = `<html>
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
			<form action="/dashboard/1" method="POST">
				<fieldset class="form-group">
					<label for "probability">Probability</label>
					<input type="number" step="0.01" min="0" max="1" class="form-control" id="probability" name="probability" placeholder="0.2">
				</fieldset>
				<fieldset class="form-group">
					<label for "frequency">Frequency</label>
					<input type="number" min="1" max="60" class="form-control" id="frequency" name="frequency" placeholder="5">
				</fieldset>
				<div class="form-group row">
					<button type="submit" class="btn btn-primary">Submit</button>
				</div>
			</form>
		</div>
	</body>
</html>
`
		})

		Context("When the service instance exists", func() {
			JustBeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com/dashboard/1", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			Context("and the service instance can be fetched", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
						AddRow("1", "https://example.com/dashboard/1", "1", 0.2, 5)
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})

				It("returns the form", func() {
					Expect(mockRecorder.Code).To(Equal(200))
					Expect(mockRecorder.Body.String()).To(Equal(response))
				})
			})

			Context("and the service instance cannot be fetched", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
				})
			})

			It("returns an error 500", func() {
				Expect(mockRecorder.Code).To(Equal(500))
			})

			Context("when the service instance does not exist in the DB", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})

				It("returns a 410", func() {
					Expect(mockRecorder.Code).To(Equal(410))
				})
			})
		})

		Context("When the service instance does not exist at all", func() {
			BeforeEach(func() {
				rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
				mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("2").WillReturnRows(rows)
				req, _ = http.NewRequest("GET", "http://example.com/v2/service_instances/2/service_bindings/2", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			It("returns a 404", func() {
				Expect(mockRecorder.Code).To(Equal(404))
			})
		})
	})

	Describe("#UpdateServiceInstance", func() {
		var (
			response     string
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
			probability  = "0.4"
			frequency    = "10"
		)

		BeforeEach(func() {
			controller = webs.CreateController(db, conf)
			mockRecorder = httptest.NewRecorder()
		})

		Context("When the service instance exists", func() {
			JustBeforeEach(func() {
				req, _ = http.NewRequest("POST", "http://example.com/dashboard/1", strings.NewReader(fmt.Sprintf("probability=%s&frequency=%s", probability, frequency)))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			Context("and the service instance cannot be fetched", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
				})

				It("returns an error 500", func() {
					Expect(mockRecorder.Code).To(Equal(500))
				})
			})

			Context("when the service instance does not exist in the database", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})

				It("returns a 410", func() {
					Expect(mockRecorder.Code).To(Equal(410))
				})
			})

			Context("and the service instance can be fetched", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
						AddRow("1", "https://example.com/dashboard/1", "1", 0.2, 5)
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})
				Context("When probability is invalid", func() {
					BeforeEach(func() {
						response = `<html>
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
						probability = "3"
					})

					AfterEach(func() {
						probability = "0.4"
					})

					It("returns an error page", func() {
						Expect(mockRecorder.Code).To(Equal(400))
						Expect(mockRecorder.Body.String()).To(Equal(response))
					})
				})

				Context("When frequency is invalid", func() {
					BeforeEach(func() {
						response = `<html>
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
						frequency = "0"
					})

					AfterEach(func() {
						frequency = "10"
					})

					It("returns an error page", func() {
						Expect(mockRecorder.Code).To(Equal(400))
						Expect(mockRecorder.Body.String()).To(Equal(response))
					})
				})

				Context("When probability and frequency are valid", func() {
					Context("and the service instance cannot be updated", func() {
						BeforeEach(func() {
							mock.ExpectExec("UPDATE service_instances.*").WithArgs(0.4, 10, "1").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
						})

						It("returns an error 500", func() {
							Expect(mockRecorder.Code).To(Equal(500))
						})
					})

					Context("and the service instance can be updated", func() {
						BeforeEach(func() {
							mock.ExpectExec("UPDATE service_instances.*").WithArgs(0.4, 10, "1").WillReturnResult(sqlmock.NewResult(1, 1))
							response = `<html>
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
			<p>Probability: 0.4</p>
			<p>Frequency: 10</p>
		</div>
	</body>
</html>`
						})

						It("updates the service instance", func() {
							Expect(mockRecorder.Code).To(Equal(202))
							Expect(mockRecorder.Body.String()).To(Equal(response))
						})
					})
				})
			})
		})

		Context("When the service instance does not exist", func() {
			BeforeEach(func() {
				rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
				mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("2").WillReturnRows(rows)
			})

			JustBeforeEach(func() {
				req, _ = http.NewRequest("POST", "http://example.com/dashboard/2", strings.NewReader(fmt.Sprintf("probability=0.2&frequency=5", probability, frequency)))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			It("returns the form", func() {
				Expect(mockRecorder.Code).To(Equal(410))
				Expect(mockRecorder.Body.String()).To(Equal(""))
			})
		})
	})

	Describe("#GetServiceInstance", func() {
		var (
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			controller = webs.CreateController(db, conf)
			mockRecorder = httptest.NewRecorder()
		})

		Context("When the service instance exists", func() {
			JustBeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com/v2/service_instances/1", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			Context("and the database does not return an error", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
						AddRow("1", "https://example.com/dashboard/1", "1", 0.2, 5)
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnRows(rows)
				})

				It("returns dashboard URL, probability and frequency", func() {
					Expect(mockRecorder.Code).To(Equal(200))
					Expect(mockRecorder.Body.String()).To(Equal(`{"dashboard_url":"https://example.com/dashboard/1","probability":0.2,"frequency":5}`))
				})
			})

			Context("when the database returns an error", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("1").WillReturnError(fmt.Errorf("An error has occured: %s", "DB error"))
				})

				It("returns an error 500", func() {
					Expect(mockRecorder.Code).To(Equal(500))
				})
			})
		})

		Context("When the service instance does not exist", func() {
			BeforeEach(func() {
				rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
				mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("2").WillReturnRows(rows)
				req, _ = http.NewRequest("GET", "http://example.com/v2/service_instances/2", nil)
				Router(controller).ServeHTTP(mockRecorder, req)
			})

			It("returns a 404 not found", func() {
				Expect(mockRecorder.Code).To(Equal(404))
			})
		})
	})

	Describe("#CreateServiceInstance", func() {
		var (
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
			instance     sharedModel.ServiceInstance
		)

		JustBeforeEach(func() {
			mockRecorder = httptest.NewRecorder()
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
			Router(controller).ServeHTTP(mockRecorder, req)
		})

		BeforeEach(func() {
			reqJSON := `{
  "organization_guid": "org-guid-here",
  "plan_id":           "plan-guid-here",
  "service_id":        "service-guid-here",
  "space_guid":        "space-guid-here"
 }`
			req, _ = http.NewRequest("PUT", "http://example.com/v2/service_instances/test", bytes.NewReader([]byte(reqJSON)))
		})

		Context("when VCAP_APPLICATION is set", func() {
			BeforeEach(func() {
				vcapApplicationJSON := `{"application_name": "test", "application_uris": ["example.com"]}`
				os.Setenv("VCAP_APPLICATION", vcapApplicationJSON)
			})

			AfterEach(func() {
				os.Unsetenv("VCAP_APPLICATION")
			})

			Context("and PROBABILITY is set", func() {
				Context("and it is set via ENV", func() {
					BeforeEach(func() {
						os.Setenv("PROBABILITY", "0.2")
						controller = webs.CreateController(db, conf)
					})

					AfterEach(func() {
						os.Unsetenv("PROBABILITY")
					})

					Context("and FREQUENCY is set via ENV", func() {
						BeforeEach(func() {
							os.Setenv("FREQUENCY", "5")
						})

						AfterEach(func() {
							os.Unsetenv("FREQUENCY")
						})

						It("Adds a instance and returns dashboard URL, probability and frequency", func() {
							Expect(mockRecorder.Code).To(Equal(201))
							Expect(mockRecorder.Body.String()).To(Equal(`{"dashboard_url":"https://example.com/dashboard/test","probability":0.2,"frequency":5}`))
						})

						Context("and request json is invalid", func() {
							BeforeEach(func() {
								reqJSON := `{
  "organization_guid": "org-guid-her
  "space_guid":        "space-guid-here"
 }`
								req, _ = http.NewRequest("PUT", "http://example.com/v2/service_instances/test", bytes.NewReader([]byte(reqJSON)))
							})

							It("returns an error 500", func() {
								Expect(os.Getenv("VCAP_APPLICATION")).To(Equal(`{"application_name": "test", "application_uris": ["example.com"]}`))
								Expect(os.Getenv("PROBABILITY")).To(Equal("0.2"))
								Expect(os.Getenv("FREQUENCY")).To(Equal("5"))
								Expect(mockRecorder.Code).To(Equal(500))
							})
						})
					})

					Context("and FREQUENCY is unset", func() {
						It("returns an error 500", func() {
							Expect(os.Getenv("VCAP_APPLICATION")).To(Equal(`{"application_name": "test", "application_uris": ["example.com"]}`))
							Expect(os.Getenv("PROBABILITY")).To(Equal("0.2"))
							Expect(os.Getenv("FREQUENCY")).To(Equal(""))
							Expect(mockRecorder.Code).To(Equal(500))
						})
					})
				})

				Context("and it is set via conf", func() {
					Context("and FREQUENCY is set via conf", func() {

						BeforeEach(func() {
							conf = &config.Config{DefaultProbability: 0.2, DefaultFrequency: 5}
							controller = webs.CreateController(db, conf)
						})

						It("Adds a instance and returns dashboard URL, probability and frequency", func() {
							Expect(mockRecorder.Code).To(Equal(201))
							Expect(mockRecorder.Body.String()).To(Equal(`{"dashboard_url":"https://example.com/dashboard/test","probability":0.2,"frequency":5}`))
						})
					})
				})
			})

			Context("and PROBABILITY is unset", func() {
				BeforeEach(func() {
					controller = webs.CreateController(db, conf)
				})

				It("returns an error 500", func() {
					Expect(os.Getenv("VCAP_APPLICATION")).To(Equal(`{"application_name": "test", "application_uris": ["example.com"]}`))
					Expect(os.Getenv("PROBABILITY")).To(Equal(""))
					Expect(mockRecorder.Code).To(Equal(500))
				})
			})
		})

		Context("when VCAP_APPLICATION is unset", func() {
			BeforeEach(func() {
				controller = webs.CreateController(db, conf)
			})

			It("returns an error 500", func() {
				Expect(os.Getenv("VCAP_APPLICATION")).To(Equal(""))
				Expect(mockRecorder.Code).To(Equal(500))
			})
		})
	})

	Describe("#Bind", func() {
		var (
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
			binding      sharedModel.ServiceBinding
		)

		BeforeEach(func() {
			mockRecorder = httptest.NewRecorder()
		})

		JustBeforeEach(func() {
			controller = webs.CreateController(db, conf)
			Router(controller).ServeHTTP(mockRecorder, req)
		})

		Context("when the request body is invalid", func() {
			BeforeEach(func() {
				reqJSON := `{
  "plan_id":      "plan-guid-h
  "service_id":   "service-guiere"
 }`
				req, _ = http.NewRequest("PUT", "http://example.com/v2/service_instances/test/service_bindings/1", bytes.NewReader([]byte(reqJSON)))
			})

			It("returns an error 500", func() {
				Expect(mockRecorder.Code).To(Equal(500))
			})
		})

		Context("when the request body is valid", func() {
			BeforeEach(func() {
				reqJSON := `{
  "plan_id":      "plan-guid-here",
  "service_id":   "service-guid-here",
  "app_guid":     "app-guid-here"
 }`
				req, _ = http.NewRequest("PUT", "http://example.com/v2/service_instances/test/service_bindings/1", bytes.NewReader([]byte(reqJSON)))
			})

			Context("and the service instance exists", func() {
				Context("and the service instance can be fetched", func() {
					BeforeEach(func() {
						rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
							AddRow("test", "https://example.com/dashboard/1", "1", 0.2, 5)
						mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("test").WillReturnRows(rows)
					})

					Context("and the binding appID is not nil", func() {
						var (
							planID     = "1"
							instanceID = "test"
							bindingID  = "1"
							appID      = "app-guid-here"
						)

						BeforeEach(func() {
							binding.ID = bindingID
							binding.ServicePlanID = planID
							binding.ServiceInstanceID = instanceID
							binding.AppID = appID
						})

						Context("and the service binding can be added", func() {
							BeforeEach(func() {
								mock.ExpectExec("INSERT INTO service_bindings").WithArgs(bindingID, appID, planID, instanceID, "").WillReturnResult(sqlmock.NewResult(1, 1))
							})

							It("Adds a binding and returns credentials", func() {
								Expect(mockRecorder.Code).To(Equal(201))
								Expect(mockRecorder.Body.String()).To(Equal(`{"credentials":{"probability":0.2,"frequency":5}}`))
							})
						})

						Context("and the service binding cannot be added", func() {
							BeforeEach(func() {
								mock.ExpectExec("INSERT INTO service_bindings").WithArgs(bindingID, appID, planID, instanceID, "").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
							})

							It("returns an error 500", func() {
								Expect(mockRecorder.Code).To(Equal(500))
							})
						})
					})

					Context("and the binding appID is nil", func() {
						BeforeEach(func() {
							reqJSON := `{
  "plan_id":      "plan-guid-here",
  "service_id":   "service-guid-here",
  "app_guid":     ""
 }`
							req, _ = http.NewRequest("PUT", "http://example.com/v2/service_instances/test/service_bindings/1", bytes.NewReader([]byte(reqJSON)))
						})

						It("returns an error 500", func() {
							Expect(mockRecorder.Code).To(Equal(500))
						})
					})
				})

				Context("and the service instance cannot be fetched", func() {
					BeforeEach(func() {
						mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("test").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))
					})

					It("returns an error 500", func() {
						Expect(mockRecorder.Code).To(Equal(500))
					})
				})
			})

			Context("When the service instance does not exist", func() {
				BeforeEach(func() {
					rows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"})
					mock.ExpectQuery("^SELECT (.+) FROM service_instances WHERE id=").WithArgs("test").WillReturnRows(rows)
					controller = webs.CreateController(db, conf)
					Router(controller).ServeHTTP(mockRecorder, req)
				})

				It("Adds a binding and returns credentials", func() {
					Expect(mockRecorder.Code).To(Equal(404))
				})
			})
		})
	})

	Describe("#Catalog", func() {
		var (
			controller   *webs.Controller
			req          *http.Request
			mockRecorder *httptest.ResponseRecorder
		)

		Context("When catalog path is set by ENV", func() {
			BeforeEach(func() {
				controller = webs.CreateController(db, conf)
				req, _ = http.NewRequest("GET", "http://example.com/v2/catalog", nil)
				mockRecorder = httptest.NewRecorder()
			})

			AfterEach(func() {
				os.Unsetenv("CATALOG_PATH")
			})

			Context("and the catalog file can be found", func() {
				Context("and the catalog file is valid", func() {
					BeforeEach(func() {
						os.Setenv("CATALOG_PATH", "fixtures/valid")
					})

					It("Returns the catalog", func() {
						controller.Catalog(mockRecorder, req)
						Expect(mockRecorder.Code).To(Equal(200))
						Expect(mockRecorder.Body.String()).To(ContainSubstring(`"id":"chaos-galago"`))
					})
				})

				Context("and the catalog file is invalid", func() {
					BeforeEach(func() {
						os.Setenv("CATALOG_PATH", "fixtures/invalid")
					})

					It("Returns an error 500", func() {
						controller.Catalog(mockRecorder, req)
						Expect(mockRecorder.Code).To(Equal(500))
					})
				})
			})

			Context("and the catalog file cannot be found", func() {
				BeforeEach(func() {
					os.Setenv("CATALOG_PATH", "fixtures/not_found")
				})

				It("Returns an error 500", func() {
					controller.Catalog(mockRecorder, req)
					Expect(mockRecorder.Code).To(Equal(500))
				})
			})
		})

		Context("When the catalog path is set by conf", func() {
			BeforeEach(func() {
				conf = &config.Config{CatalogPath: "fixtures/valid"}
				controller = webs.CreateController(db, conf)
				req, _ = http.NewRequest("GET", "http://example.com/v2/catalog", nil)
				mockRecorder = httptest.NewRecorder()
			})

			It("Returns the catalog", func() {
				controller.Catalog(mockRecorder, req)
				Expect(mockRecorder.Code).To(Equal(200))
				Expect(mockRecorder.Body.String()).To(ContainSubstring(`"id":"chaos-galago"`))
			})
		})

		Context("When catalog path is not set", func() {
			BeforeEach(func() {
				controller = webs.CreateController(db, conf)
				req, _ = http.NewRequest("GET", "http://example.com/v2/catalog", nil)
				mockRecorder = httptest.NewRecorder()
			})

			It("Returns an error 500", func() {
				controller.Catalog(mockRecorder, req)
				Expect(mockRecorder.Code).To(Equal(500))
			})
		})
	})
})
