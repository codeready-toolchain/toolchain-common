package test

// T our minimal testing interface for our custom assertions
type T interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
	Fail()
	Fatalf(format string, args ...interface{})
}
