package main

import (
	"github.com/protobuf-thrift/utils/logger"
)

func main() {
	runner, err := NewRunner()
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Convert started, please wait.")

	err = runner.Run()
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Convert succeeded.")
}
