package broker

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Jeffail/benthos/v3/internal/component/metrics"
	"github.com/Jeffail/benthos/v3/internal/component/output"
	"github.com/Jeffail/benthos/v3/internal/message"
)

var _ output.Streamed = &RoundRobin{}

func TestRoundRobinDoubleClose(t *testing.T) {
	oTM, err := NewRoundRobin([]output.Streamed{}, metrics.Noop())
	if err != nil {
		t.Error(err)
		return
	}

	// This shouldn't cause a panic
	oTM.CloseAsync()
	oTM.CloseAsync()
}

//------------------------------------------------------------------------------

func TestBasicRoundRobin(t *testing.T) {
	nMsgs := 1000

	outputs := []output.Streamed{}
	mockOutputs := []*MockOutputType{
		{},
		{},
		{},
	}

	for _, o := range mockOutputs {
		outputs = append(outputs, o)
	}

	readChan := make(chan message.Transaction)
	resChan := make(chan error)

	oTM, err := NewRoundRobin(outputs, metrics.Noop())
	if err != nil {
		t.Error(err)
		return
	}
	if err = oTM.Consume(readChan); err != nil {
		t.Error(err)
		return
	}

	for i := 0; i < nMsgs; i++ {
		content := [][]byte{[]byte(fmt.Sprintf("hello world %v", i))}
		select {
		case readChan <- message.NewTransaction(message.QuickBatch(content), resChan):
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for broker send")
			return
		}

		go func() {
			var ts message.Transaction
			select {
			case ts = <-mockOutputs[i%3].TChan:
				if !bytes.Equal(ts.Payload.Get(0).Get(), content[0]) {
					t.Errorf("Wrong content returned %s != %s", ts.Payload.Get(0).Get(), content[0])
				}
			case <-mockOutputs[(i+1)%3].TChan:
				t.Errorf("Received message in wrong order: %v != %v", i%3, (i+1)%3)
				return
			case <-mockOutputs[(i+2)%3].TChan:
				t.Errorf("Received message in wrong order: %v != %v", i%3, (i+2)%3)
				return
			case <-time.After(time.Second):
				t.Errorf("Timed out waiting for broker propagate")
				return
			}

			select {
			case ts.ResponseChan <- nil:
			case <-time.After(time.Second):
				t.Errorf("Timed out responding to broker")
				return
			}
		}()

		select {
		case res := <-resChan:
			if res != nil {
				t.Errorf("Received unexpected errors from broker: %v", res)
			}
		case <-time.After(time.Second):
			t.Errorf("Timed out responding to broker")
			return
		}
	}

	oTM.CloseAsync()
	if err := oTM.WaitForClose(time.Second * 10); err != nil {
		t.Error(err)
	}
}

//------------------------------------------------------------------------------

func BenchmarkBasicRoundRobin(b *testing.B) {
	nOutputs, nMsgs := 3, b.N

	outputs := []output.Streamed{}
	mockOutputs := []*MockOutputType{}

	for i := 0; i < nOutputs; i++ {
		mockOutputs = append(mockOutputs, &MockOutputType{})
		outputs = append(outputs, mockOutputs[i])
	}

	readChan := make(chan message.Transaction)
	resChan := make(chan error)

	oTM, err := NewRoundRobin(outputs, metrics.Noop())
	if err != nil {
		b.Error(err)
		return
	}
	if err = oTM.Consume(readChan); err != nil {
		b.Error(err)
		return
	}

	content := [][]byte{[]byte("hello world")}

	b.StartTimer()

	for i := 0; i < nMsgs; i++ {
		readChan <- message.NewTransaction(message.QuickBatch(content), resChan)
		ts := <-mockOutputs[i%3].TChan
		ts.ResponseChan <- nil
		res := <-resChan
		if res != nil {
			b.Errorf("Received unexpected errors from broker: %v", res)
		}
	}

	b.StopTimer()
}

//------------------------------------------------------------------------------
