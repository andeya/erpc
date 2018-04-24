package utils

import (
	"fmt"
)

func ExampleToJsonStr() {
	s := `<>&{}""`
	fmt.Printf("%s\n", ToJsonStr([]byte(s), true))
	fmt.Printf("%s\n", ToJsonStr([]byte(s), false))
	// Output:
	// "\u003c\u003e\u0026{}\"\""
	// "<>&{}\"\""
}
