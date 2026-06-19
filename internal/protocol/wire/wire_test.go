package wire

import "testing"

func TestWriteFixedString(t *testing.T) {
	t.Parallel()

	dst := make([]byte, 10)
	WriteFixedString(dst, "hi")

	want := []byte("hi        ")
	for i, b := range dst {
		if b != want[i] {
			t.Fatalf("byte[%d] = %q, want %q", i, b, want[i])
		}
	}
}

func TestWriteFixedStringOverflow(t *testing.T) {
	t.Parallel()

	dst := make([]byte, 4)
	WriteFixedString(dst, "hello")
	if string(dst) != "hell" {
		t.Fatalf("got %q, want hell", string(dst))
	}
}

func TestReadFixedString(t *testing.T) {
	t.Parallel()

	src := []byte("hello\x00\x00 ")
	got := ReadFixedString(src)
	if got != "hello" {
		t.Fatalf("got %q, want hello", got)
	}
}

func TestRoundTripFixedString(t *testing.T) {
	t.Parallel()

	const value = "Solar"
	buf := make([]byte, 64)
	WriteFixedString(buf, value)
	got := ReadFixedString(buf)
	if got != value {
		t.Fatalf("round trip got %q, want %q", got, value)
	}
}
