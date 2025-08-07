package tinyq

import (
	"net/http"

	"github.com/sfi2k7/blueweb"
)

func (s *tinyQ) Serve() error {

	err := s.Open()

	if err != nil {
		panic(err)
	}

	if s.port == 0 {
		s.port = 8080
	}

	ispaused := func() bool {
		if v, _ := s.Get("is_paused"); v == "Y" {
			return true
		}
		return false
	}

	togglepauseq := func(pause bool) {
		if pause {
			s.Set("is_paused", "Y")
		} else {
			s.Set("is_paused", "N")
		}
	}

	sendOk := func(ctx *blueweb.Context, message string) {
		ctx.Status(http.StatusOK)
		ctx.String(message)
	}

	web := blueweb.NewRouter()
	systemapi := web.Group("/system")

	systemapi.Get("/pause", func(ctx *blueweb.Context) {
		togglepauseq(true)
		sendOk(ctx, "ok")
	})

	systemapi.Get("/resume", func(ctx *blueweb.Context) {
		togglepauseq(false)
		sendOk(ctx, "ok")
	})

	systemapi.Get("/status", func(ctx *blueweb.Context) {
		pausestatus := ispaused()
		if pausestatus {
			sendOk(ctx, "paused")
		} else {
			sendOk(ctx, "running")
		}
	})

	tinyqapi := web.Group("/tinyq")
	tinyqapi.Get("/ack", func(ctx *blueweb.Context) {
		item := ctx.Query("item")
		if len(item) == 0 {
			ctx.Status(http.StatusOK)
			return
		}

		if err := s.RemoveItem(item); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}

		sendOk(ctx, "acked")
	})

	tinyqapi.Get("/push", func(ctx *blueweb.Context) {
		item := ctx.Query("item")
		if len(item) == 0 {
			ctx.Status(http.StatusOK)
			return
		}

		if err := s.Push(item); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}

		sendOk(ctx, "ok")
	})

	tinyqapi.Get("/pop", func(ctx *blueweb.Context) {
		if ispaused() {
			sendOk(ctx, "paused")
			return
		}

		channel := ctx.Query("channel")
		if channel == "" {
			ctx.Status(http.StatusBadRequest)
			return
		}

		items, err := s.Pop(channel)
		if err != nil {
			sendOk(ctx, "error")
			return
		}

		if len(items) == 0 {
			sendOk(ctx, "empty")
			return
		}

		sendOk(ctx, items[0])
	})

	web.Config().SetDev(true).SetPort(s.port).StopOnInterrupt()
	return web.StartServer()
}
