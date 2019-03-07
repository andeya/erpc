package tp

// Message types
const (
	TypeUndefined byte = 0
	TypeCall      byte = 1
	TypeReply     byte = 2 // reply to call
	TypePush      byte = 3
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
	default:
		return "Undefined"
	}
}
