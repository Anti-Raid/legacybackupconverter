package main

import (
	"os"

	"github.com/anti-raid/legacybackupconverter/converter"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		panic("Usage: legacybackupconverter <path to legacy backup> <path to output file> [<password>]")
	}

	legacyBackupPath := args[1]
	outputFilePath := args[2]
	var password string
	if len(args) > 3 {
		password = args[3]
	}

	fileBytes, err := os.ReadFile(legacyBackupPath)
	if err != nil {
		panic(err)
	}

	data, err := converter.ConvertFile(fileBytes, password)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(outputFilePath, data, 0644)
	if err != nil {
		panic(err)
	}
}
