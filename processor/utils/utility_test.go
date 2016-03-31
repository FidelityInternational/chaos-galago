package utils_test

import (
	"fmt"
	"github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/github.com/DATA-DOG/go-sqlmock"
	"github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/github.com/cloudfoundry-community/go-cfclient"
	. "github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/FidelityInternational/chaos-galago/processor/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/FidelityInternational/chaos-galago/processor/model"
	"github.com/FidelityInternational/chaos-galago/processor/utils"
	"math/rand"
	"os"
	"time"
)

var _ = Describe("#ShouldRun", func() {
	BeforeEach(func() {
		rand.Seed(time.Now().UTC().UnixNano())
	})

	Context("When the probability is 0", func() {
		It("Returns false", func() {
			Expect(utils.ShouldRun(0)).To(BeFalse())
		})
	})

	Context("When the probability is 1", func() {
		It("Returns true", func() {
			Expect(utils.ShouldRun(1)).To(BeTrue())
		})
	})
})

var _ = Describe("#IsAppHealthy", func() {
	Context("When all app instances are running", func() {
		It("returns true", func() {
			appInstances := make(map[string]cfclient.AppInstance)
			appInstances["0"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["1"] = cfclient.AppInstance{State: "RUNNING"}
			Expect(utils.IsAppHealthy(appInstances)).To(BeTrue())
		})
	})

	Context("When not all app instances are running", func() {
		It("returns false", func() {
			appInstances := make(map[string]cfclient.AppInstance)
			appInstances["0"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["1"] = cfclient.AppInstance{State: "STARTING"}
			Expect(utils.IsAppHealthy(appInstances)).To(BeFalse())
		})
	})
})

var _ = Describe("#PickAppInstance", func() {
	Context("When there are no app instances", func() {
		It("Returns an app index of 0", func() {
			appInstances := make(map[string]cfclient.AppInstance)
			Expect(utils.PickAppInstance(appInstances)).To(Equal(0))
		})
	})

	Context("When there is 1 app instance", func() {
		It("Returns an app index of 0", func() {
			appInstances := make(map[string]cfclient.AppInstance)
			appInstances["0"] = cfclient.AppInstance{State: "RUNNING"}
			Expect(utils.PickAppInstance(appInstances)).To(Equal(0))
		})
	})

	Context("When there is 6 app instances", func() {
		It("Returns an app index between 0 and 5", func() {
			appInstances := make(map[string]cfclient.AppInstance)
			appInstances["0"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["1"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["2"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["3"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["4"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["5"] = cfclient.AppInstance{State: "RUNNING"}
			Expect(utils.PickAppInstance(appInstances)).To(BeNumerically(">=", 0))
			Expect(utils.PickAppInstance(appInstances)).To(BeNumerically("<=", 5))
		})
	})

	Context("When there is 10 app instances, starting at 10", func() {
		It("Returns an integer value of the app index", func() {
			appInstances := make(map[string]cfclient.AppInstance)
			appInstances["10"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["11"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["12"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["13"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["14"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["15"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["16"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["17"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["18"] = cfclient.AppInstance{State: "RUNNING"}
			appInstances["19"] = cfclient.AppInstance{State: "RUNNING"}
			Expect(utils.PickAppInstance(appInstances)).To(BeNumerically(">=", 10))
			Expect(utils.PickAppInstance(appInstances)).To(BeNumerically("<=", 19))
		})
	})
})

var _ = Describe("#GetBoundApps", func() {
	Context("when services instances can be fetched", func() {
		Context("and service bindings can be fetched", func() {
			It("Gets all bound apps that require processing", func() {
				db, mock, err := sqlmock.New()
				if err != nil {
					fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
					os.Exit(1)
				}
				defer db.Close()

				instanceRows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
					AddRow("1", "example.com/1", "1", 0.2, 5).
					AddRow("2", "example.com/2", "1", 0.4, 10).
					AddRow("3", "example.com/3", "1", 0, 10)

				bindingRows := sqlmock.NewRows([]string{"id", "appID", "servicePlanID", "serviceInstanceID", "lastProcessed"}).
					AddRow("1", "1", "1", "1", "2014-11-12T10:31:20Z").
					AddRow("2", "2", "1", "2", "2014-11-12T10:34:20Z").
					AddRow("3", "3", "1", "2", "").
					AddRow("4", "4", "1", "3", "2014-11-12T10:34:20Z").
					AddRow("5", "5", "1", "4", "2014-11-12T10:34:20Z")

				mock.ExpectQuery("^SELECT (.+) FROM service_instances$").WillReturnRows(instanceRows)
				mock.ExpectQuery("^SELECT (.+) FROM service_bindings$").WillReturnRows(bindingRows)

				services := utils.GetBoundApps(db)
				Expect(services).To(HaveLen(3))
				Expect(services).To(ContainElement(model.Service{AppID: "1", LastProcessed: "2014-11-12T10:31:20Z", Probability: 0.2, Frequency: 5}))
				Expect(services).To(ContainElement(model.Service{AppID: "2", LastProcessed: "2014-11-12T10:34:20Z", Probability: 0.4, Frequency: 10}))
				Expect(services).To(ContainElement(model.Service{AppID: "3", LastProcessed: "", Probability: 0.4, Frequency: 10}))
			})
		})

		Context("and service bindings cannot be fetched", func() {
			It("returns an empty services object", func() {
				db, mock, err := sqlmock.New()
				if err != nil {
					fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
					os.Exit(1)
				}
				defer db.Close()

				instanceRows := sqlmock.NewRows([]string{"id", "dashboardURL", "planID", "probability", "frequency"}).
					AddRow("1", "example.com/1", "1", 0.2, 5).
					AddRow("2", "example.com/2", "1", 0.4, 10).
					AddRow("3", "example.com/3", "1", 0, 10)

				mock.ExpectQuery("^SELECT (.+) FROM service_instances$").WillReturnRows(instanceRows)
				mock.ExpectQuery("^SELECT (.+) FROM service_bindings$").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))

				services := utils.GetBoundApps(db)
				Expect(services).To(HaveLen(0))
			})
		})
	})

	Context("when services instances cannot be fetched", func() {
		It("returns an empty services object", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			mock.ExpectQuery("^SELECT (.+) FROM service_instances$").WillReturnError(fmt.Errorf("An error has occurred: %s", "DB error"))

			services := utils.GetBoundApps(db)
			Expect(services).To(HaveLen(0))
		})
	})
})

var _ = Describe("#ShouldProcess", func() {
	Context("When lastProcessed is in a valid format", func() {
		var timeStamp string

		BeforeEach(func() {
			layout := "2006-01-02T15:04:05Z"
			duration := time.Duration(-5) * time.Minute
			timeStamp = time.Now().UTC().Add(duration).Format(layout)
		})

		Context("and lastProcessed is older than frequency minutes", func() {
			It("returns true", func() {
				Expect(utils.ShouldProcess(5, timeStamp)).To(BeTrue())
			})
		})

		Context("and lastProcessed is newer than frequency minutes", func() {
			It("returns false", func() {
				Expect(utils.ShouldProcess(6, timeStamp)).To(BeFalse())
			})
		})
	})

	Context("When last processed is blank", func() {
		It("returns true", func() {
			Expect(utils.ShouldProcess(5, "")).To(BeTrue())
		})
	})

	Context("When lastProcessed is in an invalid format", func() {
		It("returns false", func() {
			Expect(utils.ShouldProcess(5, "not_a_date_string")).To(BeFalse())
		})
	})
})

var _ = Describe("#UpdateLastProcessed", func() {
	It("Updates the service instance", func() {
		db, mock, err := sqlmock.New()
		if err != nil {
			fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
			os.Exit(1)
		}
		defer db.Close()

		appID := "1"
		lastProcessed := "2014-11-12T10:34:20Z"

		mock.ExpectExec("UPDATE service_bindings.*").WithArgs(lastProcessed, appID).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(utils.UpdateLastProcessed(db, appID, lastProcessed)).To(BeNil())
	})

	Context("when the sql update command errors", func() {
		It("returns an error", func() {
			db, mock, err := sqlmock.New()
			if err != nil {
				fmt.Printf("\nan error '%s' was not expected when opening a stub database connection\n", err)
				os.Exit(1)
			}
			defer db.Close()

			appID := "1"
			lastProcessed := "2014-11-12T10:34:20Z"

			mock.ExpectExec("UPDATE service_bindings.*").WithArgs(lastProcessed, appID).WillReturnError(fmt.Errorf("An error has occurred: %s", "UPDATE error"))
			err = utils.UpdateLastProcessed(db, appID, lastProcessed)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("An error has occurred: UPDATE error"))
		})
	})
})

var _ = Describe("#TimeNow", func() {
	var timeNow string

	BeforeEach(func() {
		layout := "2006-01-02T15:04:05Z"
		timeNow = time.Now().UTC().Format(layout)
	})

	It("Returns a string value of the time now", func() {
		Expect(utils.TimeNow()).To(Equal(timeNow))
	})
})

var _ = Describe("LoadCFConfig", func() {
	var vcapServicesJSON string

	JustBeforeEach(func() {
		os.Setenv("VCAP_SERVICES", vcapServicesJSON)
	})

	AfterEach(func() {
		os.Unsetenv("VCAP_SERVICES")
	})

	Context("When VCAP Services is valid json", func() {
		Context("and the cf-service exists", func() {
			BeforeEach(func() {
				vcapServicesJSON = `{
  "user-provided": [
   {
    "credentials": {
    	"username":"test_user",
    	"password":"test_password",
    	"domain":"example.com",
    	"skipsslvalidation":true
    },
    "label": "user-provided",
    "name": "cf-service",
    "syslog_drain_url": "",
    "tags": []
   }
  ]
 }`
			})

			It("Returns *cfclient.Config", func() {
				config := utils.LoadCFConfig()
				Expect(config).To(Equal(&cfclient.Config{ApiAddress: "https://api.example.com",
					LoginAddress:      "https://login.example.com",
					Username:          "test_user",
					Password:          "test_password",
					SkipSslValidation: true}))
			})
		})

		Context("and the cf-service does not exist", func() {
			BeforeEach(func() {
				vcapServicesJSON = `{}`
			})

			It("returns an empty config object", func() {
				config := utils.LoadCFConfig()
				Expect(config).To(Equal(&cfclient.Config{}))
			})
		})
	})

	Context("When VCAP Services is invalid json", func() {
		BeforeEach(func() {
			vcapServicesJSON = `{
  "user-provided": [
   {
    "credentials": {
    	"username":"test_user",
    	"password":"test_password",
    	"domain":"example.com",
    	"skipsslvalidation
    "syslog_drain_url": "",
    "tags": []
   }
  ]
 }`
		})

		It("returns an empty config object", func() {
			config := utils.LoadCFConfig()
			Expect(config).To(Equal(&cfclient.Config{}))
		})
	})
})
