package local

// UnknownResult means a daemon write may have committed but independent verification failed.
// Message must be safe for task persistence and user display.
type UnknownResult struct{ Message string }

// Error returns the actionable non-secret outcome diagnostic.
func (failure *UnknownResult) Error() string { return failure.Message }
