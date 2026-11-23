package transport

import (
	"context"
	"net/rpc"
)

// RPC contexts are not serialized. The host bounds the call and the runner
// callback with its own context, then terminates the process on return.
func call(ctx context.Context, client *rpc.Client, method string, args, reply any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pending := client.Go(method, args, reply, make(chan *rpc.Call, 1))
	select {
	case <-ctx.Done():
		// Wait for the decoder before callers inspect reply. Closing the client
		// prevents a late RPC response from racing with the returned result.
		_ = client.Close()
		<-pending.Done
		return ctx.Err()
	case completed := <-pending.Done:
		return completed.Error
	}
}
