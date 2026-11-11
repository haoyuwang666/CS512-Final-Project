// TODO(Tian): Extend the bounded, per-index result slots with a UI buffer snapshot; do not introduce an unbounded segment cache.
package playback

type downloadResult struct {
	size int
	err  error
}
type bufferSlot chan downloadResult

func newSlot() bufferSlot { return make(bufferSlot, 1) }
