package advance

type tryMutex struct {
	sema chan struct{}
}

func newTryMutex() *tryMutex {
	return &tryMutex{make(chan struct{}, 1)}
}

func (tm *tryMutex) TryLock() bool {
	select {
	case tm.sema <- struct{}{}:
		return true
	default:
		return false
	}
}

func (tm *tryMutex) Lock() {
	tm.sema <- struct{}{}
}

func (tm *tryMutex) Unlock() {
	<-tm.sema
}
