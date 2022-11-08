package main

import (
	pbThrift "github.com/kevinzfb/protobuf-thrift"
	"github.com/kevinzfb/protobuf-thrift/utils/logger"
)

func main() {
	runner, err := pbThrift.NewRunner()
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
