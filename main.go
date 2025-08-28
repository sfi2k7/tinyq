package tinyq

const (
	Stringreverse = iota + 1
	StringBase64
	StringHex
	StringShiftNumbers
	StringUpperToLower
)

type TinyQ interface {
	Push(item string) error
	Pop(channel string, count ...int) ([]string, error)
	ListAllKeys(channel string) ([]string, error)
	RemoveItem(item string) error
	ListChannels() (map[string]int, error)
	PauseChannel(channel string) error
	IsChannelPaused(channel string) (bool, error)
	UnpauseChannel(channel string) error
	ClearChannel(channel string) error
	DeleteChannel(channel string) error
	Count(channel string) (int, error)
	Stats(appname string) ([]*ChannelStats, error)
	Close() error
	Open() error
	Get(bucket, key string) (string, error)
	Set(bucket, key, value string) error
	Delete(bucket, key string) error
	Inc(bucket, key string) error
}
