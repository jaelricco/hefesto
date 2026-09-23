// Package training models a logged workout: sessions, blocks, set entries,
// the ordered set elements that make a combo, and assistance.
//
// The invariant this package exists to protect: a plain set of 8 pull-ups and
// a three-element planche combo are the same shape. One set entry, N ordered
// elements. There is no second code path.
package training
