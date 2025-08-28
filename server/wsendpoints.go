package server

import "github.com/sfi2k7/blueweb"

func wsopen(args *blueweb.WSArgs) blueweb.WsData {
	subs.AddConnection(args.ID)
	return blueweb.WsData{"id": args.ID}
}

func wsclose(args *blueweb.WSArgs) blueweb.WsData {
	subs.RemoveConnection(args.ID)
	return nil
}

func wserror(args *blueweb.WSArgs) blueweb.WsData {
	subs.RemoveConnection(args.ID)
	return nil
}
