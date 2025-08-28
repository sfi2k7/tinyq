package server

import (
	"fmt"
	"net/http"
	"time"

	_ "embed"

	"github.com/sfi2k7/blueweb"
	"github.com/sfi2k7/tinyq"
)

// //go:embed index.html
// var IndexHTML string

func view_index(ctx *blueweb.Context) {
	ctx.View("./index.html", nil)
	// ctx.SetHeader("content-type", "text/html")
	// ctx.Write([]byte(IndexHTML))
}

// func auth_middle(ctx *blueweb.Context) bool {
// 	token := ctx.Header("TINYQ_AUTH_TOKEN")
// 	s := ctx.State.(*tinyq.TinyQ)
// 	if !s.IsRequestApproved(token) {
// 		ctx.Status(http.StatusUnauthorized)
// 		ctx.String("Unauthorized")
// 		return false
// 	}

// 	ctx.State = s
// 	return true
// }

func sendOk(ctx *blueweb.Context, message string, err ...error) {
	ctx.Status(http.StatusOK)
	if len(err) > 0 {
		ctx.String(fmt.Sprintf(`{"message": "%s", "error": "%s"}`, message, err[0].Error()))
		return
	}
	ctx.String(fmt.Sprintf(`{"message": "%s"}`, message))
}

var subs = NewSubManager()

type queuecontext struct {
	*blueweb.Context
	qm      *queuemanager
	sm      *statemanager
	Appname string
	q       tinyq.TinyQ
}

func (qc *queuecontext) sendOk(message string, err ...error) {
	qc.Status(http.StatusOK)
	if len(err) > 0 {
		if err[0] != nil {
			qc.String(fmt.Sprintf(`{"message": "%s", "error": "%s"}`, message, err[0].Error()))
			return
		}
	}

	qc.String(fmt.Sprintf(`{"message": "%s"}`, message))
}

func (s *queueServer) serve() error {

	s.isrunning = true

	if s.port == 0 {
		s.port = 8080
	}

	middle := func(fn func(*queuecontext)) blueweb.Handler {
		return func(ctx *blueweb.Context) {
			start := time.Now()
			qctx := &queuecontext{
				Context: ctx,
				qm:      s.qm,
				sm:      s.sm,
			}

			appname := ctx.Query("app")
			if len(appname) == 0 {
				appname = "default"
			}

			qctx.Appname = appname

			token := ctx.Query("token")

			if len(token) > 0 {
				fmt.Println(token)
			}

			q, err := s.qm.Get(appname)

			if err != nil {
				fmt.Println(err)
				ctx.Status(http.StatusInternalServerError)
				return
			}

			qctx.q = q

			fn(qctx)

			fmt.Println(ctx.URL().Path, time.Since(start))
		}
	}

	web := blueweb.NewRouter()

	web.Ws("/tinyq/ws", func(args *blueweb.WSArgs) blueweb.WsData {
		fmt.Println(args)
		if args.EventType == blueweb.WsEventOpen {
			return wsopen(args)
		}

		if args.EventType == blueweb.WsEventClose {
			return wsclose(args)
		}

		if args.EventType == blueweb.WsEventError {
			return wserror(args)
		}

		// var appname = args.Body.String("app")
		// var domain = args.Body.String("domain")

		// if domain == "kv" {
		// 	if len(appname) == 0 {
		// 		return blueweb.WsData{"error": "appname is required"}
		// 	}

		// 	cmd := args.Body.String("cmd")
		// 	key := args.Body.String("key")
		// 	value := args.Body.String("value")

		// 	if len(cmd) == 0 {
		// 		return blueweb.WsData{"error": "cmd is required"}
		// 	}

		// 	if len(key) == 0 {
		// 		return blueweb.WsData{"error": "key is required"}
		// 	}

		// 	if cmd == "get" {
		// 		q, err := s.qm.Get(appname)
		// 		if err != nil {
		// 			return blueweb.WsData{"error": err.Error()}
		// 		}

		// 		res, err := crud_endpoint_implment(ctx, cmd, key, "")
		// 		if err != nil {
		// 			return blueweb.WsData{"error": err.Error()}
		// 		}

		// 		return blueweb.WsData{"status": "success", "value": res}
		// 	}

		// 	if cmd == "set" {
		// 		q, err := s.qm.Get(appname)
		// 		if err != nil {
		// 			return blueweb.WsData{"error": err.Error()}
		// 		}
		// 		res, err := crud_endpoint_implment(q, cmd, key, value)
		// 		if err != nil {
		// 			return blueweb.WsData{"error": err.Error()}
		// 		}

		// 		return blueweb.WsData{"status": "success", "value": res}
		// 	}

		// 	if cmd == "delete" {
		// 		q, err := s.qm.Get(appname)
		// 		if err != nil {
		// 			return blueweb.WsData{"error": err.Error()}
		// 		}
		// 		res, err := crud_endpoint_implment(q, cmd, key, "")
		// 		if err != nil {
		// 			return blueweb.WsData{"error": err.Error()}
		// 		}
		// 		return blueweb.WsData{"status": "success", "value": res}
		// 	}

		// 	return blueweb.WsData{"error": "invalid command"}
		// }

		return nil
	})

	tinyqapi := web.Group("/tinyq")

	tinyqapi.Get("/", view_index)
	tinyqapi.Get("/crud/:cmd/:key", middle(crud_endpoint))
	tinyqapi.Get("/push", middle(push_endpoint))
	tinyqapi.Get("/pop", middle(pop_endpoint))
	tinyqapi.Get("/channels", middle(channels_endpoint))
	// tinyqapi.Get("/app/secure", middle(app_secure_endpoint))
	// tinyqapi.Get("/app/open", middle(app_open_endpoint))

	tinyqapi.Get("/channels/pause", middle(channels_pause_endpoint))
	tinyqapi.Get("/channels/resume", middle(channels_resume_endpoint))
	tinyqapi.Get("/channels/status", middle(channels_status_endpoint))
	tinyqapi.Get("/channels/delete", middle(channels_delete_endpoint))
	tinyqapi.Get("/channels/clear", middle(channels_clear_endpoint))
	tinyqapi.Get("/channels/lock", middle(channel_lock_endpoint))
	tinyqapi.Get("/channels/unlock", middle(channel_unlock_endpoint))
	tinyqapi.Get("/channels/lockstatus", middle(channel_lock_status_endpoint))

	tinyqapi.Get("/stats", middle(stats_endpoint))
	tinyqapi.Get("/databases", middle(databases_endpoint))

	// tinyqapi.After(func(ctx *blueweb.Context) bool {
	// 	ctx.State = nil
	// 	ctx.SetHeader("TQ_CALL_DURATION", time.Since(ctx.Get("start").(time.Time)).String())
	// 	return true
	// })

	adminapi := tinyqapi.Group("/admin")
	adminapi.Get("/", func(ctx *blueweb.Context) {
		ctx.Status(http.StatusOK)
		ctx.String("Admin Page")
	})

	web.Config().SetDev(s.logging).SetPort(s.port).StopOnInterrupt()
	fmt.Println("Server Started")

	return web.StartServer()
}
