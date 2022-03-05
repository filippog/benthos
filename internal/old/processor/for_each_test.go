package processor

import (
	"reflect"
	"testing"

	"github.com/Jeffail/benthos/v3/internal/component/metrics"
	"github.com/Jeffail/benthos/v3/internal/log"
	"github.com/Jeffail/benthos/v3/internal/manager/mock"
	"github.com/Jeffail/benthos/v3/internal/message"
	"github.com/stretchr/testify/assert"
)

//------------------------------------------------------------------------------

func TestForEachEmpty(t *testing.T) {
	conf := NewConfig()
	conf.Type = "for_each"

	proc, err := New(conf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	exp := [][]byte{
		[]byte("foo bar baz"),
	}
	msgs, res := proc.ProcessMessage(message.QuickBatch(exp))
	if res != nil {
		t.Fatal(res)
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachBasic(t *testing.T) {
	encodeConf := NewConfig()
	encodeConf.Type = TypeBloblang
	encodeConf.Bloblang = `root = if batch_index() == 0 { content().encode("base64") }`

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, encodeConf)

	proc, err := New(conf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("Zm9vIGJhciBiYXo="),
		[]byte("MSAyIDMgNA=="),
		[]byte("aGVsbG8gZm9vIHdvcmxk"),
	}
	msgs, res := proc.ProcessMessage(message.QuickBatch(parts))
	if res != nil {
		t.Fatal(res)
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachFilterSome(t *testing.T) {
	filterConf := NewConfig()
	filterConf.Type = TypeBloblang
	filterConf.Bloblang = `root = if !content().contains("foo") { deleted() }`

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, filterConf)

	proc, err := New(conf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("foo bar baz"),
		[]byte("hello foo world"),
	}
	msgs, res := proc.ProcessMessage(message.QuickBatch(parts))
	if res != nil {
		t.Fatal(res)
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachMultiProcs(t *testing.T) {
	encodeConf := NewConfig()
	encodeConf.Type = TypeBloblang
	encodeConf.Bloblang = `root = if batch_index() == 0 { content().encode("base64") }`

	filterConf := NewConfig()
	filterConf.Type = TypeBloblang
	filterConf.Bloblang = `root = if !content().contains("foo") { deleted() }`

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, filterConf, encodeConf)

	proc, err := New(conf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("foo bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello foo world"),
	}
	exp := [][]byte{
		[]byte("Zm9vIGJhciBiYXo="),
		[]byte("aGVsbG8gZm9vIHdvcmxk"),
	}
	msgs, res := proc.ProcessMessage(message.QuickBatch(parts))
	if res != nil {
		t.Fatal(res)
	}

	if len(msgs) != 1 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
	if act := message.GetAllBytes(msgs[0]); !reflect.DeepEqual(exp, act) {
		t.Errorf("Wrong results: %s != %s", act, exp)
	}
}

func TestForEachFilterAll(t *testing.T) {
	filterConf := NewConfig()
	filterConf.Type = TypeBloblang
	filterConf.Bloblang = `root = if !content().contains("foo") { deleted() }`

	conf := NewConfig()
	conf.Type = "for_each"
	conf.ForEach = append(conf.ForEach, filterConf)

	proc, err := New(conf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	parts := [][]byte{
		[]byte("bar baz"),
		[]byte("1 2 3 4"),
		[]byte("hello world"),
	}
	msgs, res := proc.ProcessMessage(message.QuickBatch(parts))
	assert.NoError(t, res)
	if len(msgs) != 0 {
		t.Errorf("Wrong count of result msgs: %v", len(msgs))
	}
}
