package config

import (
	"github.com/BurntSushi/toml"
)

// Config is the main configuration struct.
type Config struct {
	Console Console
	Lists   []List
	Net     Net
}

// Net is the configuration struct for the baps3d net server.
type Net struct {
	// Enabled toggles whether the net server is enabled.
	Enabled bool
	// Host is the TCP host:port string for the net server.
	Host string
	// Log toggles whether the net server logs to stderr.
	Log bool
}

// List is the configuration struct for a baps3d list node.
type List struct {
	// Player is the TCP host:port string for the mounted playd instance.
	Player string
}

// Console is the configuration struct for the baps3d console.
type Console struct {
	// Enabled toggles whether the console is enabled.
	Enabled bool
}

// Parse reads a TOML config from cfile.
func Parse(cfile string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(cfile, &conf)
	if err != nil {
		return Config{}, err
	}
	return conf, nil
}
