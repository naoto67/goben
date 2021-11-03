// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package context defines the Context type, which carries deadlines,
// cancellation signals, and other request-scoped values across API boundaries
// and between processes.
//
// パッケージコンテキストは、期限を伴う
// キャンセルシグナル、プロセス間およびAPI境界を越えたその他のリクエストスコープの値コンテキストタイプを定義します。
//
// Incoming requests to a server should create a Context, and outgoing
// calls to servers should accept a Context. The chain of function
// calls between them must propagate the Context, optionally replacing
// it with a derived Context created using WithCancel, WithDeadline,
// WithTimeout, or WithValue. When a Context is canceled, all
// Contexts derived from it are also canceled.
//
// サーバーへの着信要求はコンテキストを作成する必要があり、サーバーへの発信呼び出しはコンテキストを受け入れる必要があります。それらの間の関数呼び出しのチェーンは、コンテキストを伝播する必要があり、オプションで、WithCancel、WithDeadline、WithTimeout、またはWithValueを使用して作成された派生コンテキストに置き換えます。コンテキストがキャンセルされると、そのコンテキストから派生したすべてのコンテキストもキャンセルされます。
//
// The WithCancel, WithDeadline, and WithTimeout functions take a
// Context (the parent) and return a derived Context (the child) and a
// CancelFunc. Calling the CancelFunc cancels the child and its
// children, removes the parent's reference to the child, and stops
// any associated timers. Failing to call the CancelFunc leaks the
// child and its children until the parent is canceled or the timer
// fires. The go vet tool checks that CancelFuncs are used on all
// control-flow paths.
//
// WithCancel、WithDeadline、およびWithTimeout関数は、Context（親）を受け取り、派生したContext（子）とCancelFuncを返します。 CancelFuncを呼び出すと、子とその子がキャンセルされ、親の子への参照が削除され、関連するタイマーがすべて停止します。 CancelFuncを呼び出さないと、親がキャンセルされるかタイマーが起動するまで、子とその子がリークします。 go vetツールは、CancelFuncsがすべての制御フローパスで使用されていることを確認します。
//
// Programs that use Contexts should follow these rules to keep interfaces
// consistent across packages and enable static analysis tools to check context
// propagation:
//
// コンテキストを使用するプログラムは、パッケージ間でインターフェイスの一貫性を維持し、静的分析ツールがコンテキストの伝播をチェックできるようにするために、次のルールに従う必要があります。
//
// Do not store Contexts inside a struct type; instead, pass a Context
// explicitly to each function that needs it. The Context should be the first
// parameter, typically named ctx:
//
// 構造体タイプ内にコンテキストを格納しないでください。代わりに、コンテキストを必要とする各関数に明示的にコンテキストを渡します。 Contextは、通常ctxという名前の最初のパラメーターである必要があります。
//
// 	func DoSomething(ctx context.Context, arg Arg) error {
// 		// ... use ctx ...
// 	}
//
// Do not pass a nil Context, even if a function permits it. Pass context.TODO
// if you are unsure about which Context to use.
//
// 関数で許可されている場合でも、nilコンテキストを渡さないでください。使用するコンテキストがわからない場合は、context.TODOを渡します。
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
//
// コンテキスト値は、プロセスとAPIを転送するリクエストスコープのデータにのみ使用し、オプションのパラメーターを関数に渡すためには使用しません
//
// The same Context may be passed to functions running in different goroutines;
// Contexts are safe for simultaneous use by multiple goroutines.
//
// 同じコンテキストが、異なるゴルーチンで実行されている関数に渡される場合があります。
// コンテキストは、複数のゴルーチンが同時に使用しても安全です。
//
// See https://blog.golang.org/context for example code for a server that uses
// Contexts.
package context

import (
	"errors"
	"internal/reflectlite"
	"sync"
	"sync/atomic"
	"time"
)

// A Context carries a deadline, a cancellation signal, and other values across
// API boundaries.
// コンテキストは、期限、キャンセルシグナル、およびその他の値を伝達します
//
// Context's methods may be called by multiple goroutines simultaneously.
// コンテキストのメソッドは、複数のゴルーチンによって同時に呼び出される場合があります。
//
type Context interface {
	// Deadline returns the time when work done on behalf of this context
	// should be canceled. Deadline returns ok==false when no deadline is
	// set. Successive calls to Deadline return the same results.
	// Deadlineは、このコンテキストに代わって作業が行われた時間を返します
	// キャンセルする必要があります。期限がない場合、期限はok == falseを返します
	// 設定。 Deadlineを連続して呼び出すと、同じ結果が返されます。
	Deadline() (deadline time.Time, ok bool)

	// Done returns a channel that's closed when work done on behalf of this
	// context should be canceled. Done may return nil if this context can
	// never be canceled. Successive calls to Done return the same value.
	// The close of the Done channel may happen asynchronously,
	// after the cancel function returns.
	//
	// Doneは、このコンテキストに代わって実行された作業をキャンセルする必要があるときに閉じられたチャネルを返します。このコンテキストをキャンセルできない場合、Doneはnilを返す可能性があります。 Doneを連続して呼び出すと、同じ値が返されます。キャンセル機能が戻った後、Doneチャネルのクローズが非同期で発生する場合があります。
	//
	// WithCancel arranges for Done to be closed when cancel is called;
	// WithDeadline arranges for Done to be closed when the deadline
	// expires; WithTimeout arranges for Done to be closed when the timeout
	// elapses.
	//
	// WithCancelは、cancelが呼び出されたときにDoneが閉じられるように調整します。
	// WithDeadlineは、期限が切れたときにDoneを閉じるように手配します。
	// WithTimeoutは、タイムアウトが経過したときにDoneが閉じられるように調整します。
	//
	// Done is provided for use in select statements:
	//
	//  // Stream generates values with DoSomething and sends them to out
	//  // until DoSomething returns an error or ctx.Done is closed.
	//  func Stream(ctx context.Context, out chan<- Value) error {
	//  	for {
	//  		v, err := DoSomething(ctx)
	//  		if err != nil {
	//  			return err
	//  		}
	//  		select {
	//  		case <-ctx.Done():
	//  			return ctx.Err()
	//  		case out <- v:
	//  		}
	//  	}
	//  }
	//
	// See https://blog.golang.org/pipelines for more examples of how to use
	// a Done channel for cancellation.
	Done() <-chan struct{}

	// If Done is not yet closed, Err returns nil.
	// If Done is closed, Err returns a non-nil error explaining why:
	// Canceled if the context was canceled
	// or DeadlineExceeded if the context's deadline passed.
	// After Err returns a non-nil error, successive calls to Err return the same error.
	//
	// Doneがまだ閉じられていない場合、Errはnilを返します。
	// Doneが閉じている場合、Errは次の理由を説明するnil以外のエラーを返します。
	// コンテキストがキャンセルされた場合はキャンセルされました
	// または、コンテキストの期限が過ぎた場合はDeadlineExceeded。
	// Errがnil以外のエラーを返した後、Errを連続して呼び出すと同じエラーが返されます。
	Err() error

	// Value returns the value associated with this context for key, or nil
	// if no value is associated with key. Successive calls to Value with
	// the same key returns the same result.
	//
	// Valueは、キーのこのコンテキストに関連付けられている値を返します。キーに関連付けられている値がない場合は、nilを返します。同じキーを使用してValueを連続して呼び出すと、同じ結果が返されます。
	//
	// Use context values only for request-scoped data that transits
	// processes and API boundaries, not for passing optional parameters to
	// functions.
	//
	// コンテキスト値は、プロセスとAPI境界を通過するリクエストスコープのデータにのみ使用し、オプションのパラメーターを関数に渡すためには使用しません。
	//
	// A key identifies a specific value in a Context. Functions that wish
	// to store values in Context typically allocate a key in a global
	// variable then use that key as the argument to context.WithValue and
	// Context.Value. A key can be any type that supports equality;
	// packages should define keys as an unexported type to avoid
	// collisions.
	//
	// キーは、コンテキスト内の特定の値を識別します。 Contextに値を格納したい関数は、通常、グローバル変数にキーを割り当て、そのキーをcontext.WithValueおよびContext.Valueへの引数として使用します。キーは、同等性をサポートする任意のタイプにすることができます。
	// パッケージは、衝突を回避するために、キーをエクスポートされていないタイプとして定義する必要があります。
	//
	// Packages that define a Context key should provide type-safe accessors
	// for the values stored using that key:
	//
	// 	// Package user defines a User type that's stored in Contexts.
	// 	package user
	//
	// 	import "context"
	//
	// 	// User is the type of value stored in the Contexts.
	// 	type User struct {...}
	//
	// 	// key is an unexported type for keys defined in this package.
	// 	// This prevents collisions with keys defined in other packages.
	// 	type key int
	//
	// 	// userKey is the key for user.User values in Contexts. It is
	// 	// unexported; clients use user.NewContext and user.FromContext
	// 	// instead of using this key directly.
	// 	var userKey key
	//
	// 	// NewContext returns a new Context that carries value u.
	// 	func NewContext(ctx context.Context, u *User) context.Context {
	// 		return context.WithValue(ctx, userKey, u)
	// 	}
	//
	// 	// FromContext returns the User value stored in ctx, if any.
	// 	func FromContext(ctx context.Context) (*User, bool) {
	// 		u, ok := ctx.Value(userKey).(*User)
	// 		return u, ok
	// 	}
	Value(key interface{}) interface{}
}

// Canceled is the error returned by Context.Err when the context is canceled.
var Canceled = errors.New("context canceled")

// DeadlineExceeded is the error returned by Context.Err when the context's
// deadline passes.
var DeadlineExceeded error = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string { return "context deadline exceeded" }

// これなんの、メソッドなのか
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }

// An emptyCtx is never canceled, has no values, and has no deadline. It is not
// struct{}, since vars of this type must have distinct addresses.
//
// このタイプの変数には個別のアドレスが必要なため、struct {}
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key interface{}) interface{} {
	return nil
}

func (e *emptyCtx) String() string {
	switch e {
	case background:
		return "context.Background"
	case todo:
		return "context.TODO"
	}
	return "unknown empty Context"
}

var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)

// Background returns a non-nil, empty Context. It is never canceled, has no
// values, and has no deadline. It is typically used by the main function,
// initialization, and tests, and as the top-level Context for incoming
// requests.
// バックグラウンドは、nil以外の空のコンテキストを返します。キャンセルされることはなく、値も期限もありません。これは通常、メイン関数、初期化、テストによって使用され、着信要求のトップレベルのコンテキストとして使用されます
func Background() Context {
	return background
}

// TODO returns a non-nil, empty Context. Code should use context.TODO when
// it's unclear which Context to use or it is not yet available (because the
// surrounding function has not yet been extended to accept a Context
// parameter).
//
// TODOは、nil以外の空のコンテキストを返します。使用するコンテキストが不明な場合、またはまだ使用できない場合（周囲の関数がContextパラメーターを受け入れるように拡張されていないため）、コードはcontext.TODOを使用する必要があります。
func TODO() Context {
	return todo
}

// A CancelFunc tells an operation to abandon its work.
// A CancelFunc does not wait for the work to stop.
// A CancelFunc may be called by multiple goroutines simultaneously.
// After the first call, subsequent calls to a CancelFunc do nothing.
// CancelFuncは、操作にその作業を中止するように指示します。
// CancelFuncは、作業が停止するのを待ちません。
// CancelFuncは、複数のゴルーチンによって同時に呼び出される場合があります。
// 最初の呼び出しの後、CancelFuncへの後続の呼び出しは何もしません。
type CancelFunc func()

// WithCancel returns a copy of parent with a new Done channel. The returned
// context's Done channel is closed when the returned cancel function is called
// or when the parent context's Done channel is closed, whichever happens first.
//
// WithCancelは、新しいDoneチャネルを持つ親のコピーを返します。
// 返されたキャンセル関数が呼び出されたとき、または親コンテキストのDoneチャネルが閉じられたときのいずれか早い方で、返されたコンテキストのDoneチャネルが閉じられます。
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}

// newCancelCtx returns an initialized cancelCtx.
func newCancelCtx(parent Context) cancelCtx {
	return cancelCtx{Context: parent}
}

// goroutines counts the number of goroutines ever created; for testing.
var goroutines int32

// propagateCancel arranges for child to be canceled when parent is.
func propagateCancel(parent Context, child canceler) {
	done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

	select {
	case <-done:
		// parent is already canceled
		child.cancel(false, parent.Err())
		return
	default:
	}

	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			// parent cancelCtxのchildrenに自分の情報を入れる
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		atomic.AddInt32(&goroutines, +1)
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}

// &cancelCtxKey is the key that a cancelCtx returns itself for.
var cancelCtxKey int

// parentCancelCtx returns the underlying *cancelCtx for parent.
// It does this by looking up parent.Value(&cancelCtxKey) to find
// the innermost enclosing *cancelCtx and then checking whether
// parent.Done() matches that *cancelCtx. (If not, the *cancelCtx
// has been wrapped in a custom implementation providing a
// different done channel, in which case we should not bypass it.)
//
// parentCancelCtxは、親の基になる* cancelCtxを返します。
// これを行うには、parent.Value（＆cancelCtxKey）を検索して、最も内側を囲む* cancelCtxを見つけ、parent.Done（）がその* cancelCtxと一致するかどうかを確認します。 （そうでない場合、* cancelCtxは、別の完了チャネルを提供するカスタム実装にラップされています。その場合、それをバイパスしないでください。）
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
	done := parent.Done()
	if done == closedchan || done == nil {
		return nil, false
	}
	p, ok := parent.Value(&cancelCtxKey).(*cancelCtx)
	if !ok {
		return nil, false
	}
	pdone, _ := p.done.Load().(chan struct{})
	if pdone != done {
		return nil, false
	}
	return p, true
}

// removeChild removes a context from its parent.
func removeChild(parent Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}

// A canceler is a context type that can be canceled directly. The
// implementations are *cancelCtx and *timerCtx.
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}

// closedchan is a reusable closed channel.
var closedchan = make(chan struct{})

func init() {
	// 常にclosedchanから struct{}が取得できるようにする？
	// なんのためなのかはわからない
	close(closedchan)
}

// A cancelCtx can be canceled. When canceled, it also cancels any children
// that implement canceler.
// cancelCtxはキャンセルできます。キャンセルされると、キャンセラーを実装するすべての子もキャンセルされます。
type cancelCtx struct {
	Context

	mu       sync.Mutex            // protects following fields
	done     atomic.Value          // of chan struct{}, created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
}

func (c *cancelCtx) Value(key interface{}) interface{} {
	if key == &cancelCtxKey {
		return c
	}
	return c.Context.Value(key)
}

func (c *cancelCtx) Done() <-chan struct{} {
	d := c.done.Load()
	if d != nil {
		return d.(chan struct{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
}

func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

type stringer interface {
	String() string
}

func contextName(c Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}
	return reflectlite.TypeOf(c).String()
}

func (c *cancelCtx) String() string {
	return contextName(c.Context) + ".WithCancel"
}

// cancel closes c.done, cancels each of c's children, and, if
// removeFromParent is true, removes c from its parent's children.
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	d, _ := c.done.Load().(chan struct{})
	if d == nil {
		c.done.Store(closedchan)
	} else {
		close(d)
	}
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		// デッドロックが発生しないことを注意書きしてる？
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}

// WithDeadline returns a copy of the parent context with the deadline adjusted
// to be no later than d. If the parent's deadline is already earlier than d,
// WithDeadline(parent, d) is semantically equivalent to parent. The returned
// context's Done channel is closed when the deadline expires, when the returned
// cancel function is called, or when the parent context's Done channel is
// closed, whichever happens first.
// WithDeadlineは、期限がdまでになるように調整された親コンテキストのコピーを返します。親の期限がすでにdより前の場合、WithDeadline（parent、d）は意味的に親と同等です。返されるコンテキストのDoneチャネルは、期限が切れたとき、返されたキャンセル関数が呼び出されたとき、または親コンテキストのDoneチャネルが閉じられたときのいずれか早い方で閉じられます。

//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	// 期限がありかつ、デッドラインが設定値より前の場合は親のctxから作成する
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		return WithCancel(parent)
	}
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded) // deadline has already passed
		return c, func() { c.cancel(false, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		// AfterFunc dur後にfuncを実行する
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}

// A timerCtx carries a timer and a deadline. It embeds a cancelCtx to
// implement Done and Err. It implements cancel by stopping its timer then
// delegating to cancelCtx.cancel.
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) String() string {
	return contextName(c.cancelCtx.Context) + ".WithDeadline(" +
		c.deadline.String() + " [" +
		time.Until(c.deadline).String() + "])"
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err)
	if removeFromParent {
		// Remove this timerCtx from its parent cancelCtx's children.
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}

// WithTimeout returns WithDeadline(parent, time.Now().Add(timeout)).
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete:
// このコンテキストをキャンセルすると、それに関連付けられたリソースが解放されるため、このコンテキストで実行されている操作が完了するとすぐに、コードはcancelを呼び出す必要があります。
//
// 	func slowOperationWithTimeout(ctx context.Context) (Result, error) {
// 		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
// 		defer cancel()  // releases resources if slowOperation completes before timeout elapses
// 		return slowOperation(ctx)
// 	}
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

// WithValue returns a copy of parent in which the value associated with key is
// val.
// WithValueは、キーに関連付けられた値がvalである親のコピーを返します。
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
// コンテキスト値は、プロセスとAPIを転送するリクエストスコープのデータにのみ使用し、オプションのパラメーターを関数に渡すためには使用しません。
//
// The provided key must be comparable and should not be of type
// string or any other built-in type to avoid collisions between
// packages using context. Users of WithValue should define their own
// types for keys. To avoid allocating when assigning to an
// interface{}, context keys often have concrete type
// struct{}. Alternatively, exported context key variables' static
// type should be a pointer or interface.
// 提供されるキーは比較可能である必要があり、コンテキストを使用するパッケージ間の衝突を回避するために、文字列型またはその他の組み込み型であってはなりません。
// WithValueのユーザーは、キーの独自のタイプを定義する必要があります。に割り当てるときに割り当てを回避するには interface {}、コンテキストキーは多くの場合具体的なタイプstruct {}を持っています。または、エクスポートされたコンテキストキー変数の静的タイプは、ポインターまたはインターフェイスである必要があります。
func WithValue(parent Context, key, val interface{}) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	// パニックが起きるので注意
	if key == nil {
		panic("nil key")
	}
	// Comparableは、このタイプの値が比較可能かどうかを報告します。
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}

// A valueCtx carries a key-value pair. It implements Value for that key and
// delegates all other calls to the embedded Context.
type valueCtx struct {
	Context
	key, val interface{}
}

// stringify tries a bit to stringify v, without using fmt, since we don't
// want context depending on the unicode tables. This is only used by
// *valueCtx.String().
func stringify(v interface{}) string {
	switch s := v.(type) {
	case stringer:
		return s.String()
	case string:
		return s
	}
	return "<not Stringer>"
}

func (c *valueCtx) String() string {
	return contextName(c.Context) + ".WithValue(type " +
		reflectlite.TypeOf(c.key).String() +
		", val " + stringify(c.val) + ")"
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}
