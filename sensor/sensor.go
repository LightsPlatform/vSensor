/*
 * +===============================================
 * | Author:        Parham Alvani <parham.alvani@gmail.com>
 * |
 * | Creation Date: 02-02-2018
 * |
 * | File Name:     sensor/sensor.go
 * +===============================================
 */

package sensor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/LightsPlatform/vSensor/generators"
	log "github.com/sirupsen/logrus"
)

// Sensor represents virtual sensor that
// only generate random data with given generator
type Sensor struct {
	id   int
	Name string `json:"name"`

	Buffer chan Data `json:"-"`
	quit   chan struct{}

	gen generators.Generator
}

// Data represents sensor data that contains
// time and value
type Data struct {
	Time  time.Time
	Value interface{}
}

// New creates new sensor and store its user given script
func New(name string, script []byte) (*Sensor, error) {
	// Store user script
	path := os.TempDir() + string(os.PathSeparator) + "sensor-%s.py"
	f, err := os.Create(fmt.Sprintf(path, name))
	if err != nil {
		return nil, err
	}
	f.Write(script)

	return &Sensor{
		Name:   name,
		Buffer: make(chan Data, 1024),
		quit:   make(chan struct{}, 0),

		gen: generators.UniformGenerator{Timeslot: 1 * time.Second},
	}, nil
}

// Stop stops running sensor
func (s *Sensor) Stop() {
	s.quit <- struct{}{}

	close(s.quit)
	close(s.Buffer)
}

// Run runs sensor, running sensor generate data using
// user given script.
// it is a blocking function so run it in new thread
func (s *Sensor) Run() {
	g := s.gen.Generate()

	for {
		select {
		case c := <-g:
			for i := 0; i < c; i++ {
				pathf := os.TempDir() + string(os.PathSeparator) + "sensor-%s.py"
				cmd := exec.Command("runtime.py", fmt.Sprintf(pathf, s.Name))

				// run
				value, err := cmd.Output()
				if err != nil {
					if err, ok := err.(*exec.ExitError); ok {
						log.Errorf("%s: %s", err.Error(), err.Stderr)
						continue
					}
				}

				d := Data{
					Time: time.Now(),
				}

				if err := json.Unmarshal(value, &d.Value); err == nil {
					log.Infoln(d)
					s.Buffer <- d
				} else {
					log.Errorf("%s", err)
				}

			}
		case <-s.quit:
			return
		}
	}
}
