package test

// T our minimal testing interface for our custom assertions
type T interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
	Fatalf(format string, args ...interface{})
}
