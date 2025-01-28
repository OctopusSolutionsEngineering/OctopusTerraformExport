package collections

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestSafeErrorSlice_Append(t *testing.T) {
	ss := &SafeErrorSlice{}
	err := errors.New("test error")
	ss.Append(err)

	if len(ss.slice) != 1 {
		t.Errorf("expected slice length 1, got %d", len(ss.slice))
	}

	if ss.slice[0] != err {
		t.Errorf("expected error %v, got %v", err, ss.slice[0])
	}
}

func TestSafeErrorSlice_GetCopy(t *testing.T) {
	ss := &SafeErrorSlice{}
	err1 := errors.New("test error 1")
	err2 := errors.New("test error 2")
	ss.Append(err1)
	ss.Append(err2)

	copy := ss.GetCopy()

	if !reflect.DeepEqual(copy, ss.slice) {
		t.Errorf("expected copy %v, got %v", ss.slice, copy)
	}

	if &copy == &ss.slice {
		t.Errorf("expected different slices, got same")
	}
}

func TestSafeErrorSlice_ConcurrentAppend(t *testing.T) {
	ss := &SafeErrorSlice{}
	numGoroutines := 100
	numErrors := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < numErrors; j++ {
				ss.Append(errors.New("test error"))
			}
		}(i)
	}

	wg.Wait()

	expectedLength := numGoroutines * numErrors
	if len(ss.slice) != expectedLength {
		t.Errorf("expected slice length %d, got %d", expectedLength, len(ss.slice))
	}
}
