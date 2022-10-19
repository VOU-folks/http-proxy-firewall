package json_rpc

type Error struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}
