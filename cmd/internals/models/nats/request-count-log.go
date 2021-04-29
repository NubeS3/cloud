package nats

type ReqLog struct {
	Event
	Method   string `json:"method"`
	Req      string `json:"req"`
	SourceIp string `json:"source_ip"`
}

type AuthReqLog struct {
	Event
	Method   string `json:"method"`
	Req      string `json:"req"`
	SourceIp string `json:"source_ip"`
	UserId   string `json:"user_id"`
}

type AccessKeyReqLog struct {
	Event
	Method   string `json:"method"`
	Req      string `json:"req"`
	SourceIp string `json:"source_ip"`
	Key      string `json:"key"`
}

type SignedReqLog struct {
	Event
	Method   string `json:"method"`
	Req      string `json:"req"`
	SourceIp string `json:"source_ip"`
	Public   string `json:"public"`
}
