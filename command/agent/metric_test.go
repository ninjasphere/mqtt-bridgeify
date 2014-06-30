package agent

import (
	"log"

	. "launchpad.net/gocheck"
)

type LoadMetricSuite struct {
	metricService *MetricService
}

var _ = Suite(&LoadMetricSuite{})

func (s *LoadMetricSuite) SetUpTest(c *C) {
	s.metricService = CreateMetricService()
}

func (s *LoadMetricSuite) TestMetricCall(c *C) {

	req := s.metricService.buildMetricsRequest()

	log.Printf("%++v", req)

	c.Assert(req, Not(Equals), nil)

}
