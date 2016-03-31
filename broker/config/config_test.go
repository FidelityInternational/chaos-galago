package config_test

import (
	"fmt"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/FidelityInternational/chaos-galago/broker/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/FidelityInternational/chaos-galago/broker/config"
)

var _ = Describe("#LoadConfig_#GetConfig", func() {
	It("Loads configuration from a file into a struct", func() {
		_, err := LoadConfig("fixtures/test_config.json")
		if err != nil {
			panic(fmt.Sprintf("Error loading config file [%s]...", err.Error()))
		}
		conf := GetConfig()
		Expect(conf.CatalogPath).To(Equal("test"))
		Expect(conf.DefaultProbability).To(Equal(0.4))
		Expect(conf.DefaultFrequency).To(Equal(10))
	})

	Context("When the file cannot be read", func() {
		It("returns an error", func() {
			_, err := LoadConfig("fixtures/no_config.json")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(MatchRegexp("open fixtures/no_config.json"))
		})
	})

	Context("When the file is invalid json", func() {
		It("returns an error", func() {
			_, err := LoadConfig("fixtures/invalid_config.json")
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(MatchRegexp("invalid"))
		})
	})
})
