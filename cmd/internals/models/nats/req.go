package nats

type Req struct {
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Type   string   `json:"type"`
	Data   []string `json:"data"`
}

type Res struct {
	Data      string   `json:"type"`
	ExtraData []string `json:"extra_data"`
}
