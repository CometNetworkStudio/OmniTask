package middleware

import (
	"net/url"
	"testing"
)

func TestRedactQuery(t *testing.T) {
	raw := "accessToken=secret-token&page=2&password=p%40ss&token=a&token=b&name=ok"
	got, err := url.ParseQuery(redactQuery(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("page") != "2" || got.Get("name") != "ok" {
		t.Fatalf("non-sensitive params changed: %v", got)
	}
	for _, key := range []string{"accessToken", "password", "token"} {
		values := got[key]
		if len(values) == 0 {
			t.Fatalf("missing key %s in %v", key, got)
		}
		for _, value := range values {
			if value != "***" {
				t.Fatalf("%s leaked: %v", key, values)
			}
		}
	}
	if redactQuery("") != "" {
		t.Fatal("empty query should stay empty")
	}
}
