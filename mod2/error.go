package mod2

import (
	"fmt"
	"os"
)

// PrintOnError 错误处理
func PrintOnError(err error, msg string) {
	if err != nil {
		fmt.Printf("%s: %s\n", msg, err)
	}
}

// ExitOnError 退出
func ExitOnError(err error, msg string) {
	if err != nil {
		fmt.Printf("%s: %s\n", msg, err)
		os.Exit(0)
	}
}
