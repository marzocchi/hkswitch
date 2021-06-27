package output

import (
	"bytes"
	"testing"
)

func TestPrefix(t *testing.T) {
	if is, want := Prefix("hello", 12, " | "), "hello        | "; is != want {
		t.Fatalf("is = %v, want = %v", is, want)
	}
}

func TestWithPrefix(t *testing.T) {
	buf := &bytes.Buffer{}

	pw := WithPrefix(buf, "prfx: ")

	_, _ = pw.Write([]byte("hello\nw"))
	_, _ = pw.Write([]byte("orld"))

	want := "prfx: hello\nprfx: world"
	if is := buf.String(); is != want {
		t.Fatalf("is = %q, want = %q", is, want)
	}
}
