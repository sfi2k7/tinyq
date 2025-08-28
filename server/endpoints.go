package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

func channels_delete_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error")
		return
	}

	if islocked, err := ctx.sm.IsChannelLocked(ctx.Appname, channel); err != nil {
		ctx.sendOk("error")
		return
	} else if islocked {
		ctx.sendOk("locked")
		return
	}

	if err := ctx.q.DeleteChannel(channel); err != nil {
		ctx.sendOk("error")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "delete_channel", channel)

	ctx.sendOk("ok")
}

func channel_lock_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error")
		return
	}

	if err := ctx.sm.LockChannel(ctx.Appname, channel); err != nil {
		ctx.sendOk("error")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "lock_channel", channel)
	ctx.sendOk("ok")
}

func channel_unlock_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error")
		return
	}

	if err := ctx.sm.UnlockChannel(ctx.Appname, channel); err != nil {
		ctx.sendOk("error")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "unlock_channel", channel)
	ctx.sendOk("ok")
}

func channel_lock_status_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error")
		return
	}

	status, err := ctx.sm.IsChannelLocked(ctx.Appname, channel)
	if err != nil {
		ctx.sendOk("error")
		return
	}

	var str string
	if status {
		str = "locked"
	} else {
		str = "unlocked"
	}

	ctx.sm.AddStat(ctx.Appname, "channel_status", channel)
	ctx.sendOk(str)
}

func channels_clear_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error")
		return
	}

	if islocked, err := ctx.sm.IsChannelLocked(ctx.Appname, channel); err != nil {
		ctx.sendOk("error")
		return
	} else if islocked {
		ctx.sendOk("locked")
		return
	}

	if err := ctx.q.ClearChannel(channel); err != nil {
		ctx.sendOk("error")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "clear_channel", channel)
	ctx.sendOk("ok")
}

func databases_endpoint(ctx *queuecontext) {
	dirs, err := os.ReadDir("ctx.q.Rootpath")
	if err != nil {
		ctx.sendOk(`{"error": "error getting databases"}`)
	}

	var dbs []string
	for _, de := range dirs {
		if strings.HasSuffix(de.Name(), ".db") {
			dbs = append(dbs, strings.TrimSuffix(de.Name(), ".db"))
		}
	}

	ctx.sm.AddStat(ctx.Appname, "list_databases", "")

	ctx.Json(dbs)
}

func push_endpoint(ctx *queuecontext) {
	item := ctx.Query("item")
	if len(item) == 0 {
		ctx.sendOk("error", errors.New("item is missing"))
		return
	}

	if err := ctx.q.Push(item); err != nil {
		ctx.sendOk("ok")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "push", item)
	ctx.sendOk("ok")
}

func pop_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	count, _ := ctx.QueryInt("count")
	if channel == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}

	// paused, err := ctx.q.IsChannelPaused(channel)
	paused, err := ctx.sm.IsChannelPaused(ctx.Appname, channel)
	if err != nil {
		fmt.Println(err)
		ctx.sendOk("error", err)
		return
	}

	if paused {
		fmt.Println("paused")
		ctx.sendOk("paused")
		return
	}

	items, err := ctx.q.Pop(channel, count)
	if err != nil {
		fmt.Println(err)
		ctx.sendOk("error", err)
		return
	}

	if len(items) == 0 {
		ctx.sendOk("empty")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "pop", channel)

	ctx.sendOk(items[0])
}

func stats_endpoint(ctx *queuecontext) {

	stats, err := ctx.sm.Stats(ctx.Appname) // ctx.q.Stats(ctx.AppName)
	if err != nil {
		fmt.Println(err)
		ctx.sendOk("error", err)
		return
	}

	sort.Slice(stats, func(i int, j int) bool {
		return stats[i].Channel < stats[j].Channel
	})

	fmt.Println("sending back", stats)
	ctx.Json(stats)
}

func channels_pause_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error")
		return
	}

	if err := ctx.sm.PauseChannel(ctx.Appname, channel); err != nil {
		ctx.sendOk("error")
		return
	}

	ctx.sm.AddStat(ctx.Appname, "channel_pause", channel)
	ctx.sendOk("paused")
}

func channels_status_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error", errors.New("channel is missing"))
		return
	}

	ispaused, err := ctx.sm.IsChannelPaused(ctx.Appname, channel)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.sm.AddStat(ctx.Appname, "channel_status", channel)

	ctx.sendOk(strconv.FormatBool(ispaused))
}

func channels_resume_endpoint(ctx *queuecontext) {
	channel := ctx.Query("channel")
	if channel == "" {
		ctx.sendOk("error", errors.New("channel is missing"))
		return
	}

	if err := ctx.sm.ResumeChannel(ctx.Appname, channel); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.sm.AddStat(ctx.Appname, "channel_resume", channel)
	ctx.sendOk("unpaused")
}

func channels_endpoint(ctx *queuecontext) {
	channels, err := ctx.q.ListChannels()
	if err != nil {
		ctx.sendOk("error", err)
		return
	}

	var str = strings.Builder{}

	for channel, count := range channels {
		isPaused, _ := ctx.q.IsChannelPaused(channel)
		isLocked, _ := ctx.sm.IsChannelLocked(ctx.Appname, channel)
		str.WriteString(fmt.Sprintf(`%s|%d|%t|%t\n`, channel, count, isPaused, isLocked))
	}

	ctx.sm.AddStat(ctx.Appname, "channel_list", "")
	ctx.sendOk(str.String())
}

func crud_endpoint_implment(ctx *queuecontext, command, key, value string) (string, error) {
	if command == "set" {
		if err := ctx.q.Set("kv", key, value); err != nil {
			return "error", err
		}
		return "ok", nil
	}

	if command == "get" {
		if val, err := ctx.q.Get("kv", key); err != nil {
			return "error", err
		} else {
			return val, nil
		}
	}

	if command == "delete" {
		if err := ctx.q.Delete("kv", key); err != nil {
			return "error", err
		}
		return "ok", nil
	}

	return "error", errors.New("invalid command")
}

func crud_endpoint(ctx *queuecontext) {
	command := ctx.Params("cmd")
	key := ctx.Params("key")
	value := ctx.Query("v")

	res, err := crud_endpoint_implment(ctx, command, key, value)
	if err == nil {
		ctx.sendOk(res)
		return
	}

	ctx.sendOk(res, err)
}
