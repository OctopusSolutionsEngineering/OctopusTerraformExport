package collections

import "sync"

type SafeErrorSlice struct {
	sync.Mutex
	slice []error
}

func (ss *SafeErrorSlice) GetCopy() []error {
	ss.Lock()
	defer ss.Unlock()
	cpy := make([]error, len(ss.slice))
	copy(cpy, ss.slice)
	return cpy
}

func (ss *SafeErrorSlice) Append(val error) {
	ss.Lock()
	defer ss.Unlock()
	ss.slice = append(ss.slice, val)
}
