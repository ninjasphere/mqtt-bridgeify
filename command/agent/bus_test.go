package agent

import (
	"bytes"
	"encoding/json"
	"strings"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	. "launchpad.net/gocheck"
)

type LoadBusSuite struct {
	bus           *Bus
	connectReq    *connectRequest
	disconnectReq *disconnectRequest
	sampleJson    string
}

var _ = Suite(&LoadBusSuite{})

func (s *LoadBusSuite) SetUpTest(c *C) {

	s.bus = &Bus{}
	s.sampleJson = `{"url":"ssl://dev.ninjasphere.co","token":"123123123"}`
	s.connectReq = &connectRequest{
		Url:   "ssl://dev.ninjasphere.co",
		Token: "123123123",
	}

}

func (s *LoadBusSuite) TestEncode(c *C) {

	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(s.connectReq)
	c.Assert(err, IsNil)
	c.Assert(trim(string(buf.Bytes())), Equals, trim(s.sampleJson))

}

func (s *LoadBusSuite) TestDecode(c *C) {
	req := &connectRequest{}
	msg := mqtt.NewMessage([]byte(s.sampleJson))
	err := s.bus.decodeRequest(msg, req)
	c.Assert(err, IsNil)
	c.Assert(req, DeepEquals, s.connectReq)
}

func trim(str string) string {
	return strings.Trim(str, "\n\r")
}
