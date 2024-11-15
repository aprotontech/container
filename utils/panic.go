package utils

import (
	"errors"
	"fmt"
)

func Assert(err error, msg ...any) {
	if err != nil {
		panic(err)
	}
}

func CatchException() {
	if r := recover(); r != nil {
		fmt.Println("Recovered from panic:", r)
	}
}

func OnError(target error, action func()) {
	if r := recover(); r != nil {
		if err, ok := r.(error); ok && errors.Is(err, target) {
			action()
		}
	}
}
