package main

import (
	"context"
	"net/http"

	firebase "firebase.google.com/go"
	go_receive "github.com/Seklfreak/go-receive"
	"github.com/Seklfreak/receivenow/pkg/refresher"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	fb, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		logger.Fatal("failure initialising firebase", zap.Error(err))
	}

	fs, err := fb.Firestore(context.Background())
	if err != nil {
		logger.Fatal("failure initialising firebase firestore", zap.Error(err))
	}

	goReceive := go_receive.New(http.DefaultClient)

	refr := refresher.New(fs, goReceive)
	err = refr.Do(context.Background())
	if err != nil {
		logger.Fatal("failure refreshing items", zap.Error(err))
	}

}
