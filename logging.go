package main

import (
	"os"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
)

type centralTimeHandler struct {
	h log.Handler
}

func (h *centralTimeHandler) HandleLog(e *log.Entry) error {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return err
	}
	e.Timestamp = e.Timestamp.In(loc)
	return h.h.HandleLog(e)
}

func loggingSetup() {
	log.SetHandler(&centralTimeHandler{
		h: json.New(os.Stdout),
	})
	log.SetLevel(log.InfoLevel)
}
