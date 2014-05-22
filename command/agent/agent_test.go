package agent

import (
	. "launchpad.net/gocheck"

	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type LoadAgentSuite struct{
  agent *Agent
  topic *replaceTopic
}

var _ = Suite(&LoadAgentSuite{})

func (s *LoadAgentSuite) SetUpTest(c *C) {

  conf := &Config{}

	s.agent = createAgent(conf)
  s.topic = &replaceTopic{on: "$location/calibration", replace: "$location", with: "$cloud/location"}
}


func (s *LoadAgentSuite) TestConfig(c *C) {

  res := s.topic.updated("$location/calibration")
  exp := "$cloud/location/calibration"

  c.Assert(res, Equals, exp)


}
