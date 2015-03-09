package agent

import (
	"bytes"
	"sync"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/juju/loggo"
)

var (
	prefix = []byte(`{"batch":[`)
	suffix = []byte(`]}`)
)

// Route performs the routing of messages between two connections along with batching for certain topics.
type Route struct {
	Log         loggo.Logger
	Topic       string
	Source      string
	Counter     int
	Destination interface {
		PublishMessage(topic string, message *mqtt.Message) <-chan mqtt.Receipt
	}
	ticker *time.Ticker

	lk   sync.Mutex
	msgs [][]byte
}

// Start the route
func (r *Route) Start() {
	r.ticker = time.NewTicker(1 * time.Second)

	go r.readMsgs()
}

// Stop cancel the route
func (r *Route) Stop() {
	if r.ticker != nil {
		if err := r.Flush(); err != nil {
			r.Log.Warningf("unable to flush: %s", err)
		}

		r.ticker.Stop()
	}
}

// Flush the route, this builds a combined message and sends it through
// to the destination
func (r *Route) Flush() error {

	var combined []byte

	r.lk.Lock()

	defer r.lk.Unlock()

	combined = bytes.Join(r.msgs, []byte(","))
	combined = append(prefix, combined...)
	combined = append(combined, suffix...)

	payload := r.updateSource(combined, r.Source)
	r.Destination.PublishMessage(r.Topic, mqtt.NewMessage(payload))

	r.Log.Debugf("Route flushed %s %d", r.Topic, len(r.msgs))

	r.msgs = [][]byte{}

	return nil
}

func (r *Route) MsgHandler(src *mqtt.MqttClient, msg mqtt.Message) {

	if r.Log.IsDebugEnabled() {
		r.Log.Debugf("(%s) topic: %s len: %d", r.Source, msg.Topic(), len(msg.Payload()))
	}
	r.Counter++

	r.lk.Lock()
	defer r.lk.Unlock()

	r.msgs = append(r.msgs, msg.Payload())
}

func (r *Route) readMsgs() {
	for _ = range r.ticker.C {
		if len(r.msgs) > 0 {
			r.Flush()
		}
	}
	r.Log.Debugf("Route finished reading %s", r.Topic)
}

func (r *Route) updateSource(payload []byte, source string) []byte {

	if !bytes.Contains(payload, []byte("$mesh-source")) {
		payload = bytes.Replace(payload, []byte("{"), []byte(`{"$mesh-source":"`+source+`", `), 1)
	}

	r.Log.Debugf("msg %s", string(payload))

	return payload
}
