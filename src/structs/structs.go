package structs

type GatewayResponse struct {
	URL string `json:"url"`
}

type Message struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
	S  *int        `json:"s,omitempty"`
	T  *string     `json:"t,omitempty"`
}

type HelloMessageData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type IdentifyMessageData struct {
	Token      string `json:"token"`
	Properties Props  `json:"properties"`
	Intents    int    `json:"intents"`
}

type Props struct {
	Os      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type ReadyPayload struct {
	SessionId        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
}

type Command struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        int       `json:"type,omitempty"`
	Options     []Command `json:"options,omitempty"`
	Required    bool      `json:"required,omitempty"`
	Value       string    `json:"value,omitempty"`
}

type Interaction struct {
	Data  interface{} `json:"data"`
	Token string      `json:"token"`
}

type InteractionData struct {
	Name    string      `json:"name"`
	Options interface{} `json:"options"`
}

type InteractionDataOptions struct {
	Name  string      `json:"name"`
	Type  int         `json:"type"`
	Value interface{} `json:"value"`
}

type User struct {
	Id            string `json:"id,omitempty"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
}
