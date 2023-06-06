package main

type config struct {
	Web    webConfig     `yaml:"web"`
	Meters []meterConfig `yaml:"meters"`
}

type webConfig struct {
	Address           string `yaml:"address"`
	DisableRequestLog bool   `yaml:"disable_request_log"`
}

type meterConfig struct {
	Id                  string `yaml:"id"`
	Address             string `yaml:"address"`
	ReconnectDelay      int    `yaml:"reconnect_delay"`
	ReadTimeout         int    `yaml:"read_timeout"`
	ConnectTimeout      int    `yaml:"connect_timeout"`
	DisableReceptionLog bool   `yaml:"disable_reception_log"`
	Debug               bool   `yaml:"debug"`
}
