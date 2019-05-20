package rabbit

import (
	"errors"
	"testing"
	"time"
)

func assertNilError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expecting error to be nil but got %v", err)
	}
}

func TestPubSub(t *testing.T) {
	subscribers, err := CreateSubscriber(url, "OneWallet")
	assertNilError(t, err)

	err = subscribers.AddTopics([]string{"reactor.#"})
	assertNilError(t, err)

	publisher, err := CreatePublisher(url, "OneWallet", []string{"reactor.123"})
	assertNilError(t, err)

	err = publisher.Publish([]byte("hello world"))
	assertNilError(t, err)

	select {
	case d := <-subscribers.Messages:
		{
			message := string(d.Body)
			if message != "hello world" {
				t.Fatalf("expecting message to equal to 'hello world' but got %q", message)
			}
		}
	case <-time.After(time.Second * 1):
		assertNilError(t, errors.New("timeout"))
	}

	err = subscribers.Close()
	assertNilError(t, err)
}
