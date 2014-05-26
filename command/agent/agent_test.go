package agent

import . "launchpad.net/gocheck"

type LoadAgentSuite struct {
	agent          *Agent
	topicToCloud   *replaceTopic
	topicFromCloud *replaceTopic
}

var _ = Suite(&LoadAgentSuite{})

func (s *LoadAgentSuite) SetUpTest(c *C) {

	conf := &Config{}

	s.agent = createAgent(conf)
	s.topicToCloud = &replaceTopic{on: "$location/calibration", replace: "$location", with: "$cloud/location"}
	s.topicFromCloud = &replaceTopic{on: "$location/calibration", replace: "$location", with: "$cloud/location"}
}

func (s *LoadAgentSuite) TestConfig(c *C) {

	res := s.topicToCloud.updated("$location/calibration")
	exp := "$cloud/location/calibration"

	c.Assert(res, Equals, exp)

	res = s.topicFromCloud.updated("$location/calibration")
	exp = "$cloud/location/calibration"

	c.Assert(res, Equals, exp)

}
