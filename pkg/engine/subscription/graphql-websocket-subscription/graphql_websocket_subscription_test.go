package graphql_websocket_subscription

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jensneuse/graphql-go-tools/pkg/engine/subscription"
)

func TestGraphQLWebsocketSubscriptionStream(t *testing.T) {

	server := FakeGraphQLSubscriptionServer(t)
	defer server.Close()

	host := server.Listener.Addr().String()

	//host = "localhost:4444"

	stream := New()
	ctx, cancel := context.WithCancel(context.Background())
	manager := subscription.NewManager(stream)
	defer cancel()

	manager.Run(ctx.Done())

	input := fmt.Sprintf(`{"scheme":"ws","host":"%s","path":"","body":{"query":"subscription{counter{count}}","variables":{}}}`,host)

	read := func(wg *sync.WaitGroup,tag string, trigger subscription.Trigger, then func(), messages ...string) {
		wg.Add(1)
		go func(wg *sync.WaitGroup, trigger subscription.Trigger, then func(), messages ...string) {
			defer func() {
				manager.StopTrigger(trigger)
				if then != nil {
					then()
				}
				wg.Done()
			}()
			for i := range messages {
				data, ok := trigger.Next(context.Background())
				if !ok {
					panic("want ok")
				}
				actual := string(data)
				expected := messages[i]
				assert.Equal(t, expected, actual)
			}
		}(wg, trigger, then, messages...)
	}

	t1 := manager.StartTrigger([]byte(input))
	t2 := manager.StartTrigger([]byte(input))
	t3 := manager.StartTrigger([]byte(input))
	wg := &sync.WaitGroup{}

	assert.Equal(t, int64(1), manager.TotalSubscriptions())
	assert.Equal(t, int64(3), manager.TotalSubscribers())

	read(wg,"t1", t1, func() {
		assert.Equal(t, int64(1), manager.TotalSubscriptions())
		assert.Equal(t, int64(2), manager.TotalSubscribers())
	}, `{"data":{"counter":{"count":0}}}`)
	read(wg, "t2",t2, nil, `{"data":{"counter":{"count":0}}}`, `{"data":{"counter":{"count":1}}}`, `{"data":{"counter":{"count":2}}}`)
	read(wg, "t3",t3, nil, `{"data":{"counter":{"count":0}}}`, `{"data":{"counter":{"count":1}}}`, `{"data":{"counter":{"count":2}}}`)
	wg.Wait()

	assert.Equal(t, int64(0), manager.TotalSubscriptions())
	assert.Equal(t, int64(0), manager.TotalSubscribers())

	t4 := manager.StartTrigger([]byte(input))

	assert.Equal(t, int64(1), manager.TotalSubscriptions())
	assert.Equal(t, int64(1), manager.TotalSubscribers())

	wg = &sync.WaitGroup{}

	read(wg,"t4", t4, func() {
		assert.Equal(t, int64(0), manager.TotalSubscriptions())
		assert.Equal(t, int64(0), manager.TotalSubscribers())
	}, `{"data":{"counter":{"count":0}}}`)

	wg.Wait()

	assert.Equal(t, int64(0), manager.TotalSubscriptions())
	assert.Equal(t, int64(0), manager.TotalSubscribers())
}