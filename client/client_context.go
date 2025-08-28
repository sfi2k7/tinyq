package client

import (
	"fmt"
	"strconv"
	"time"
)

type WebWorkerContext struct {
	Item    string
	ID      string
	Data    map[string]string
	Client  *WebClient
	Channel string
}

func (ctx *WebWorkerContext) RouteNoOp() string {
	return ""
}

func (ctx *WebWorkerContext) NoOp() string {
	return ""
}

func (ctx *WebWorkerContext) addpairstodata(pairs ...any) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	for i := 0; i < len(pairs); i += 2 {
		if i+1 < len(pairs) {
			k := pairs[i].(string)
			v := pairs[i+1]
			switch ft := v.(type) {
			case string:
				ctx.DataSet(k, ft)
			case int:
				ctx.DataSetInt(k, ft)
			case float64:
				ctx.DataSetFloat(k, ft)
			case bool:
				ctx.DataSetBool(k, ft)
			case time.Time:
				ctx.DataSetTime(k, ft)
			default:
				ctx.DataSet(k, fmt.Sprintf("%v", v))
			}
		}
	}
}

func (ctx *WebWorkerContext) SetProps(props ...any) {
	ctx.addpairstodata(props...)
}

func (ctx *WebWorkerContext) RouteTo(channel string, pairs ...any) string {
	if len(pairs) > 0 {
		ctx.addpairstodata(pairs...)
	}

	if len(ctx.Data) > 0 {
		data := serialize(ctx.Data)
		return fmt.Sprintf("%s.%s.%s", channel, ctx.ID, data)
	}

	return fmt.Sprintf("%s.%s", channel, ctx.ID)
}

func (ctx *WebWorkerContext) DataSet(k, v string) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	ctx.Data[k] = v
}

func (ctx *WebWorkerContext) DataGet(k string) string {
	if ctx.Data == nil {
		return ""
	}

	return ctx.Data[k]
}

func (ctx *WebWorkerContext) DataGetBool(k string) bool {
	if ctx.Data == nil {
		return false
	}

	v, ok := ctx.Data[k]
	if !ok {
		return false
	}

	return v == "true" || v == "1" || v == "yes" || v == "on" || v == "enabled"
}

func (ctx *WebWorkerContext) DataGetInt(k string) int {
	if ctx.Data == nil {
		return 0
	}

	v, ok := ctx.Data[k]
	if !ok {
		return 0
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}

	return i
}

func (ctx *WebWorkerContext) DataGetFloat(k string) float64 {
	if ctx.Data == nil {
		return 0
	}

	v, ok := ctx.Data[k]
	if !ok {
		return 0
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}

	return f
}

func (ctx *WebWorkerContext) DataGetUnixTime(k string) time.Time {
	if ctx.Data == nil {
		return time.Time{}
	}

	v, ok := ctx.Data[k]
	if !ok {
		return time.Time{}
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(int64(i), 0)
}

func (ctx *WebWorkerContext) DataHas(k string) bool {
	if ctx.Data == nil {
		return false
	}

	_, ok := ctx.Data[k]
	return ok
}

func (ctx *WebWorkerContext) DataSetBool(k string, v bool) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	ctx.Data[k] = strconv.FormatBool(v)
}

func (ctx *WebWorkerContext) DataSetFloat(k string, v float64) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	ctx.Data[k] = strconv.FormatFloat(v, 'f', -1, 64)
}

func (ctx *WebWorkerContext) DataSetInt(k string, v int) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	ctx.Data[k] = strconv.Itoa(v)
}

func (ctx *WebWorkerContext) DataSetTime(k string, v time.Time) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	ctx.Data[k] = strconv.FormatInt(v.Unix(), 10)
}

func (ctx *WebWorkerContext) DataDelete(k string) {
	if ctx.Data == nil {
		return
	}

	delete(ctx.Data, k)
}

func (ctx *WebWorkerContext) DataClear() {
	if ctx.Data == nil {
		return
	}

	for k := range ctx.Data {
		delete(ctx.Data, k)
	}
}
