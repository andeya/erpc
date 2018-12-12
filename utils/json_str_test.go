package utils

import (
	"fmt"
)

func ExampleToJSONStr() {
	s := `<>&{}""`
	fmt.Printf("%s\n", ToJSONStr([]byte(s), true))
	fmt.Printf("%s\n", ToJSONStr([]byte(s), false))
	// Output:
	// "\u003c\u003e\u0026{}\"\""
	// "<>&{}\"\""
}
