package primitive

type Devices struct {
	PushName   string `json:"pushName"`
	Platform   string `json:"platform"`
	User       string `json:"user"`
	Server     string `json:"server"`
	IsLoggedIn bool   `json:"isLoggedIn"`
}
