package teleport

type Context interface {
	Conn
}

type SvrContext interface {
	Conn
}

type cliContext struct {
	conn Conn
}

type svrContext struct {
	conn Conn
}
