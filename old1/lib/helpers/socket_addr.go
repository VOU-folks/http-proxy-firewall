package helpers

type SocketAddr struct {
	HOST    string
	PORT    string
	ADDRESS string
}

func GetSocketAddr() SocketAddr {
	HOST := GetEnv("TCP_SERVER_HOST")
	if HOST == "" {
		HOST = GetEnvOr("HOST", DEFAULT_HOST)
	}

	PORT := GetEnv("TCP_SERVER_PORT")
	if PORT == "" {
		PORT = GetEnvOr("PORT", DEFAULT_PORT)
	}

	ADDRESS := HOST + ":" + PORT

	return SocketAddr{HOST, PORT, ADDRESS}
}
