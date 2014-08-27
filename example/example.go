package main

import (
	"io"
	"log"
	"os"
	"strings"
)

type Model struct {
	Title    string
	Messages []string
}

// +erb
type Body func(writer io.Writer, model Model) (err error)

// +erb
type Messages func(writer io.Writer, messages []string) (err error)

func main() {
	model := Model{
		Title:    "Template",
		Messages: strings.Split("Let's go!", " "),
	}
	err := WriteBody(os.Stdout, model)
	if err != nil {
		log.Fatal(err)
	}
}
