package httpapi

import (
	"strings"
	"testing"

	"github.com/shourenso/vpet/internal/store"
)

func TestCleanChatMessageAcceptsOrdinaryLines(t *testing.T) {
	text, ok := cleanChatMessage("  Chào mọi người!  ")
	if !ok || text != "Chào mọi người!" {
		t.Fatalf("got %q ok=%v", text, ok)
	}
	if _, ok = cleanChatMessage(strings.Repeat("ă", 200)); !ok {
		t.Fatal("200 multi-byte runes rejected; the limit counts runes, not bytes")
	}
}

func TestCleanChatMessageRefusesSpamShapes(t *testing.T) {
	cases := map[string]string{
		"empty":      "   ",
		"too long":   strings.Repeat("a", 201),
		"wall":       "a\nb\nc\nd",
		"http link":  "ghé http://evil.example nhé",
		"https link": "HTTPS://Evil.Example",
		"bare www":   "www.evil.example",
	}
	for name, raw := range cases {
		if _, ok := cleanChatMessage(raw); ok {
			t.Fatalf("%s accepted: %q", name, raw)
		}
	}
}

func TestChatHubDeliversToLiveSubscribersOnly(t *testing.T) {
	hub := newChatHub()
	stream, stop := hub.subscribe()
	hub.publish(store.WorldMessage{ID: 1})
	select {
	case got := <-stream:
		if got.ID != 1 {
			t.Fatalf("id %d", got.ID)
		}
	default:
		t.Fatal("subscriber received nothing")
	}

	stop()
	hub.publish(store.WorldMessage{ID: 2})
	if len(hub.subs) != 0 {
		t.Fatalf("unsubscribed reader retained: %d", len(hub.subs))
	}
}

// A slow reader must never block the writer: publishing drops rather than stalling the HTTP
// handler that produced the message.
func TestChatHubDropsForABlockedSubscriber(t *testing.T) {
	hub := newChatHub()
	_, stop := hub.subscribe()
	defer stop()
	for i := int64(0); i < 100; i++ {
		hub.publish(store.WorldMessage{ID: i})
	}
}
