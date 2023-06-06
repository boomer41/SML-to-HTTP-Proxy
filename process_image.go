package main

import (
	"sync"
	"time"
)

type processImage struct {
	Meters map[string]processImageMeter `json:"meters"`
}

type processImageMeter struct {
	Connected  bool                              `json:"connected"`
	LastUpdate *time.Time                        `json:"lastUpdate"`
	Values     map[string]processImageMeterValue `json:"values"`
}

type processImageMeterValue struct {
	Value interface{} `json:"value"`
	Unit  uint8       `json:"unit"`
}

type processImageManager struct {
	image processImage
	lock  sync.Mutex
}

func newProcessImageManager(cfg *config) *processImageManager {
	v := &processImageManager{
		image: processImage{
			Meters: make(map[string]processImageMeter),
		},
	}

	for _, m := range cfg.Meters {
		v.image.Meters[m.Id] = processImageMeter{
			Connected:  false,
			LastUpdate: nil,
			Values:     make(map[string]processImageMeterValue),
		}
	}

	return v
}

func (i *processImageManager) updateMeterValues(meterId string, value processImageMeter) {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.image.Meters[meterId] = value
}

func (i *processImageManager) get() processImage {
	i.lock.Lock()
	defer i.lock.Unlock()

	return i.image
}
