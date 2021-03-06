package zdelta

import (
	"log"
	"reflect"
	"testing"
)

func TestCreateDelta(t *testing.T) {
	src := []byte("In Xanadu did Kublai Khan a stately pleasure-dome decree")
	target := []byte("In Ooo did Jake a bitchen pleasure-dome decree")
	delta, err := CreateDelta(src, target)
	log.Printf("delta = %x (%d bytes, %.0f%%)", delta, len(delta),
		float64(len(delta))/float64(len(target))*100)
	log.Printf("err = %v", err)
	if err != nil {
		t.Fatalf("CreateDelta returned %v", err)
	}

	target2, err := ApplyDelta(src, delta)
	log.Printf("target = %q (%d bytes)", target2, len(target2))
	log.Printf("err = %v", err)
	if err != nil {
		t.Fatalf("ApplyDelta returned %v", err)
	}
	if !reflect.DeepEqual(target, target2) {
		t.Fatalf("Reconstituted target is wrong")
	}
}

func TestReuseCompressor(t *testing.T) {
	src := []byte("In Xanadu did Kublai Khan a stately pleasure-dome decree")
	target := []byte("In Ooo did Jake a bitchen pleasure-dome decree")

	var c Compressor
	delta1, err := c.CreateDelta(src, target)
	if err != nil {
		t.Fatalf("CreateDelta returned %v", err)
	}
	log.Printf("Delta = %x", delta1)

	delta2, err := c.CreateDelta(src, target)
	if err != nil {
		t.Fatalf("CreateDelta returned %v", err)
	}
	log.Printf("Delta = %x", delta2)

	if !reflect.DeepEqual(delta1, delta2) {
		t.Fatalf("Incorrect delta after reset!")
	}
}
