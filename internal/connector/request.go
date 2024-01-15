package connector

type Request struct {
	Method string   `json:"method"`
	Params []string `json:"params,omitempty"`
	Id     int64    `json:"id"`
}
