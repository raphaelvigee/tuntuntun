package tuntunfwd

type Mode string

const (
	V1                 = 1
	ServerForward Mode = "server"
	ClientForward Mode = "client"
)

type Init struct {
	Version int    `json:"version"`
	Mode    Mode   `json:"mode,omitempty"`
	Addr    string `json:"addr,omitempty"`
}
