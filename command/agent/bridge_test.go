package agent

import (
	. "launchpad.net/gocheck"

	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type LoadBridgeSuite struct {
	agent *Bridge
	topic *replaceTopic
}

var _ = Suite(&LoadBridgeSuite{})

func (s *LoadBridgeSuite) SetUpTest(c *C) {

	conf := &Config{}

	s.agent = createBridge(conf)
	s.topic = &replaceTopic{on: "$location/calibration", replace: "$location", with: "$cloud/location"}
}

func (s *LoadBridgeSuite) TestConfig(c *C) {

	res := s.topic.updated("$location/calibration")
	exp := "$cloud/location/calibration"

	c.Assert(res, Equals, exp)

}
