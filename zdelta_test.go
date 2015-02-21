package zdelta

import (
	"bytes"
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

func TestWriteDelta(t *testing.T) {
	src := []byte("In Xanadu did Kublai Khan a stately pleasure-dome decree")
	target := []byte("In Ooo did Jake a bitchen pleasure-dome decree")

	var out bytes.Buffer
	err := WriteDelta(src, target, &out)
	if err != nil {
		t.Fatalf("WriteDelta returned %v", err)
	}
	delta := out.Bytes()
	log.Printf("delta = %x (%d bytes, %.0f%%)", delta, len(delta),
		float64(len(delta))/float64(len(target))*100)

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
