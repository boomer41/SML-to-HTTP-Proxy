package main

import (
	"fmt"
	"net"
	"sml-to-http/sml"
	"time"

	"golang.org/x/exp/constraints"
)

type meterManager struct {
	instances []*meterInstance
}

type meterInstance struct {
	config              meterConfig
	processImageManager *processImageManager
	processImageMeter   processImageMeter

	logger logger

	stopSignal chan interface{}
}

func newMeterManager(meters []meterConfig, image *processImageManager, log logger) *meterManager {
	m := &meterManager{
		instances: make([]*meterInstance, len(meters)),
	}

	meterLog := log.newSubLogger("meters")

	for i, meter := range meters {
		m.instances[i] = &meterInstance{
			config:              meter,
			processImageManager: image,
			processImageMeter: processImageMeter{
				Connected: false,
				Values:    make(map[string]processImageMeterValue),
			},
			logger: meterLog.newSubLogger(meter.Id),

			stopSignal: make(chan interface{}),
		}
	}

	return m
}

func (m *meterManager) run() error {
	errChan := make(chan error)

	for _, i := range m.instances {
		instance := i

		go func() {
			err := instance.run()

			if err != nil {
				return
			}

			errChan <- err
		}()
	}

	err := <-errChan

	return err
}

func (m *meterInstance) run() error {
	delay := false

	for {
		if delay {
			m.logger.Printf("waiting %d seconds before reconnect...", m.config.ReconnectDelay)
			time.Sleep(time.Duration(m.config.ReconnectDelay) * time.Second)
		}

		delay = true

		var timeout time.Duration

		if m.config.ConnectTimeout > 0 {
			timeout = time.Duration(m.config.ConnectTimeout) * time.Second
		}

		m.logger.Printf("connecting to %s...", m.config.Address)
		conn, err := net.DialTimeout("tcp", m.config.Address, timeout)

		if err != nil {
			m.logger.Printf("dial failed: %v", err)
			continue
		}

		m.logger.Printf("connection established")

		err = m.handleConnection(conn)

		if err != nil {
			m.logger.Printf("connection error: %v", err)
		}
	}
}

func (m *meterInstance) handleConnection(conn net.Conn) error {
	defer func() {
		m.processImageMeter.Connected = false
		m.processImageMeter.LastUpdate = nil
		m.processImageMeter.Values = make(map[string]processImageMeterValue)
		m.commitProcessImage()

		_ = conn.Close()
	}()

	m.processImageMeter.Connected = true
	m.commitProcessImage()

	smlReader := sml.NewReader(conn)

	for {
		if m.config.ReadTimeout > 0 {
			err := conn.SetReadDeadline(time.Now().Add(time.Duration(m.config.ReadTimeout) * time.Second))

			if err != nil {
				m.logger.Printf("failed to set read deadline: %v", err)
			}
		}

		f, err := smlReader.ReadFile()

		if err != nil {
			return err
		}

		if !m.config.DisableReceptionLog {
			if m.config.Debug {
				m.logger.Printf("received SML file:\n%s", f)
			} else {
				m.logger.Printf("received SML file")
			}
		}

		if len(f.Messages) != 3 {
			return fmt.Errorf("can only understand SML file with 3 messages, got %d", len(f.Messages))
		}

		valueMessage, ok := f.Messages[1].MessageBody.(*sml.GetListResMessageBody)

		if !ok {
			return fmt.Errorf("SML_GetList.Res message required")
		}

		procImage := m.processImageMeter

		now := time.Now()
		procImage.LastUpdate = &now

		procImage.Values = make(map[string]processImageMeterValue)

		for _, value := range valueMessage.ValList {
			err := m.mapValue(&procImage, value)

			if err != nil {
				return fmt.Errorf("failed to parse value: %v", err)
			}
		}

		m.processImageMeter = procImage
		m.commitProcessImage()
	}
}

func (m *meterInstance) commitProcessImage() {
	m.processImageManager.updateMeterValues(m.config.Id, m.processImageMeter)
}

func (m *meterInstance) mapValue(p *processImageMeter, value *sml.ListEntry) error {
	obis, err := sml.ObisToString(value.ObjName)

	if err != nil {
		return err
	}

	val := value.Value

	switch val.(type) {
	case *int8:
		val = smlScale(*val.(*int8), value.Scaler)
	case *int16:
		val = smlScale(*val.(*int16), value.Scaler)
	case *int32:
		val = smlScale(*val.(*int32), value.Scaler)
	case *int64:
		val = smlScale(*val.(*int64), value.Scaler)
	case *uint8:
		val = smlScale(*val.(*uint8), value.Scaler)
	case *uint16:
		val = smlScale(*val.(*uint16), value.Scaler)
	case *uint32:
		val = smlScale(*val.(*uint32), value.Scaler)
	case *uint64:
		val = smlScale(*val.(*uint64), value.Scaler)
	}

	p.Values[obis] = processImageMeterValue{
		Value: val,
		Unit:  value.Unit,
	}

	return nil
}

func smlScale[V constraints.Integer](v V, scaler int8) *float64 {
	i := float64(v)

	for scaler < 0 {
		i /= 10
		scaler++
	}

	for scaler > 0 {
		i *= 10
		scaler--
	}

	return &i
}
