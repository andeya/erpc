package tp

// Message types
const (
	TypeUndefined byte = 0
	TypeCall      byte = 1
	TypeReply     byte = 2 // reply to call
	TypePush      byte = 3
	TypeAuthCall  byte = 4
	TypeAuthReply byte = 5
)

// TypeText returns the message type text.
// If the type is undefined returns 'Undefined'.
func TypeText(typ byte) string {
	switch typ {
	case TypeCall:
		return "CALL"
	case TypeReply:
		return "REPLY"
	case TypePush:
		return "PUSH"
	case TypeAuthCall:
		return "AUTH_CALL"
	case TypeAuthReply:
		return "AUTH_REPLY"
	default:
		return "Undefined"
	}
}
