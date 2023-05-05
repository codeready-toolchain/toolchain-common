package test

// type FakeAuditClient struct {
// 	commonclient.AuditClient
// 	T                T
// 	MockGet          func(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error
// 	MockList         func(ctx context.Context, list runtimeclient.ObjectList, opts ...runtimeclient.ListOption) error
// 	MockCreate       func(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.CreateOption) error
// 	MockUpdate       func(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error
// 	MockPatch        func(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error
// 	MockStatusUpdate func(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.UpdateOption) error
// 	MockStatusPatch  func(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption) error
// 	MockDelete       func(ctx context.Context, logger logr.Logger, obj runtimeclient.Object, opts ...runtimeclient.DeleteOption) error
// }

// func NewFakeAuditClient(t T, initObjs ...runtime.Object) FakeAuditClient {
// 	cl := NewFakeClient(t, initObjs...)
// 	return commonclient.NewAuditClient(cl)
// }
