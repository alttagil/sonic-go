// Copyright (c) 2023 Alexander Khudich
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// NOTE: The code in this file has been adapted from the "bytes"
// package of the Go standard library
//
// The original copyright notice from the Go project for these parts is
// reproduced here:
//
// ========================================================================
// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
// ========================================================================

package sonic

import (
	"errors"
	"io"
)

// smallBufferSize is an initial allocation minimal capacity.
const smallBufferSize = 64

// ErrTooLarge is passed to panic if memory cannot be allocated to store data in a buffer.
var ErrTooLarge = errors.New("media.Buffer: too large")

// maxInt represents the maximum positive integer value.
const maxInt = int(^uint(0) >> 1)

// Buffer is a generic variable-sized buffer for storing arbitrary data types.
type Buffer[T any] struct {
	buf []T // contents are the elements buf[off : len(buf)]
	off int // read at &buf[off], write at &buf[len(buf)]
}

// NewBuffer creates and initializes a new Buffer using Type and Len
func NewBuffer[T any](initialCap int) *Buffer[T] {
	return &Buffer[T]{buf: make([]T, 0, initialCap)}
}

// Buffer returns a slice of length b.Len() holding the unread portion of the buffer.
func (b *Buffer[T]) Buffer() []T {
	return b.buf[b.off:]
}

// AvailableBuffer returns an empty buffer with b.Available() capacity.
func (b *Buffer[T]) AvailableBuffer() []T {
	return b.buf[len(b.buf):]
}

// Len returns the number of elements in the unread portion of the buffer.
func (b *Buffer[T]) Len() int {
	return len(b.buf) - b.off
}

// Cap returns the capacity of the buffer's underlying slice.
func (b *Buffer[T]) Cap() int {
	return cap(b.buf)
}

// Available returns how many elements are unused in the buffer.
func (b *Buffer[T]) Available() int {
	return cap(b.buf) - len(b.buf)
}

// isEmpty reports whether the unread portion of the buffer is empty.
func (b *Buffer[T]) isEmpty() bool {
	return len(b.buf) <= b.off
}

// RawSlice (EXPERIMENTAL) returns slice from a buffer without counter changes
func (b *Buffer[T]) RawSlice(n int) []T {
	if b.isEmpty() {
		b.Reset()
	}
	_, ok := b.tryGrowByReslice(n)
	if !ok {
		b.grow(n)
	}
	l := len(b.buf)

	ret := b.buf[l-n:l:l]
	b.buf = b.buf[:l-n]

	return ret
}

// RawLenAdd (EXPERIMENTAL) changes internal counters as if n elements were added to a buffer
func (b *Buffer[T]) RawLenAdd(n int) error {
	_, ok := b.tryGrowByReslice(n)
	if !ok {
		return errors.New("media.Buffer: RawLenAdd is out of range")
	}
	return nil
}

// Truncate discards all but the first n unread elements from the buffer.
func (b *Buffer[T]) Truncate(n int) {
	if n == 0 {
		b.Reset()
		return
	}
	if n < 0 || n > b.Len() {
		panic("media.Buffer: truncation out of range")
	}
	b.buf = b.buf[:b.off+n]
}

// Reset resets the buffer to be empty.
func (b *Buffer[T]) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

// Write appends the element v to the buffer, growing the buffer as needed.
func (b *Buffer[T]) Write(v T) error {
	m, ok := b.tryGrowByReslice(1)
	if !ok {
		m = b.grow(1)
	}
	b.buf[m] = v
	return nil
}

// WriteAt rewrites the element v in the buffer at position At
func (b *Buffer[T]) WriteAt(n int, v T) {
	if b.Len() < n {
		panic("media.Buffer: wrong position to write at")
	}
	b.buf[b.off+n] = v
}

// WriteSlice appends the elements of the slice to the buffer, growing the buffer as needed.
func (b *Buffer[T]) WriteSlice(slice []T) error {
	if len(slice) == 0 {
		return nil
	}

	m, ok := b.tryGrowByReslice(len(slice))
	if !ok {
		m = b.grow(len(slice))
	}
	copy(b.buf[m:], slice)

	return nil
}

// Read reads the next element from the buffer.
func (b *Buffer[T]) Read() (T, error) {
	if b.isEmpty() {
		var zeroValue T
		b.Reset()
		return zeroValue, io.EOF
	}
	v := b.buf[b.off]
	b.off++
	return v, nil
}

// DropSlice drops the next n elements from the buffer.
func (b *Buffer[T]) DropSlice(n int) error {
	if b.isEmpty() {
		b.Reset()
		return io.EOF
	}
	m := b.Len()
	if n > m {
		n = m
	}
	b.off += n
	return nil
}

// ReadSlice reads the next n elements from the buffer.
func (b *Buffer[T]) ReadSlice(n int) ([]T, error) {
	if b.isEmpty() {
		b.Reset()
		return nil, io.EOF
	}
	m := b.Len()
	if n > m {
		n = m
	}
	slice := b.buf[b.off : b.off+n]
	b.off += n
	return slice, nil
}

// ReadSliceAt reads slice from position at
func (b *Buffer[T]) ReadSliceAt(at int) ([]T, error) {
	if b.isEmpty() {
		b.Reset()
		return nil, io.EOF
	}

	if at < 0 || at > b.Len() {
		panic("media.Buffer: out of range")
	}

	slice := b.buf[b.off+at:]
	b.buf = b.buf[:b.off+at]

	return slice, nil
}

// GetSlice gets the next n elements from the buffer without removing them from a buffer
func (b *Buffer[T]) GetSlice(n int) ([]T, error) {
	if b.isEmpty() {
		b.Reset()
		return nil, io.EOF
	}
	m := b.Len()
	if n > m {
		n = m
	}
	slice := b.buf[b.off : b.off+n]
	return slice, nil
}

// GetSliceAtN gets a slice of N elements from a buffer position at certain position
func (b *Buffer[T]) GetSliceAtN(at, n int) ([]T, error) {
	if b.isEmpty() {
		b.Reset()
		return nil, io.EOF
	}

	if at < 0 || at+n < 0 || at+n > b.Len() {
		panic("media.Buffer: out of range")
	}
	slice := b.buf[b.off+at : b.off+at+n]

	return slice, nil
}

// MoveTo reads data from n position to the end of the original buffer and writes it to dest buffer
func (b *Buffer[T]) MoveTo(a *Buffer[T], n int) error {
	if b.isEmpty() {
		return nil
	}
	s, err := b.ReadSlice(n)
	if err != nil {
		return err
	}
	return a.WriteSlice(s)
}

// MoveAllTo transfers all elements from the current Buffer to another Buffer.
// If the current Buffer is empty, it returns nil. It reads a slice from
// the current Buffer and writes it to the specified destination Buffer (a).
func (b *Buffer[T]) MoveAllTo(a *Buffer[T]) error {
	if b.isEmpty() {
		return nil
	}
	s, err := b.ReadSlice(b.Len())
	if err != nil {
		return err
	}
	return a.WriteSlice(s)
}

// CopyTo gets data from n position to the end of the original buffer and writes it to dest buffer
func (b *Buffer[T]) CopyTo(dest *Buffer[T], n int) error {
	if b.isEmpty() {
		return nil
	}
	s, err := b.GetSlice(n)
	if err != nil {
		return nil
	}
	return dest.WriteSlice(s)
}

// At peeks the element of the Buffer at n position
func (b *Buffer[T]) At(n int) (T, error) {
	if len(b.buf)-b.off < n {
		var zeroValue T
		return zeroValue, io.EOF
	}
	return b.buf[b.off+n], nil
}

// Grow grows the buffer's capacity to guarantee space for another n elements.
func (b *Buffer[T]) Grow(n int) {
	if n < 0 {
		panic("media.Buffer.Grow: negative count")
	}
	m, ok := b.tryGrowByReslice(n)
	if !ok {
		m = b.grow(n)
	}
	b.buf = b.buf[:m]
}

// tryGrowByReslice is an inlineable version of grow for the fast-case where the
// internal buffer only needs to be resliced.
func (b *Buffer[T]) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); n <= cap(b.buf)-l {
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

// growSlice is a utility function for growing slices.
func growSlice[T any](b []T, n int) []T {
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	c := len(b) + n
	if c < 2*cap(b) {
		c = 2 * cap(b)
	}
	b2 := append([]T(nil), make([]T, c)...)
	copy(b2, b)
	return b2[:len(b)]
}

// grow grows the buffer to guarantee space for n more elements.
func (b *Buffer[T]) grow(n int) int {
	m := b.Len()
	if m == 0 && b.off != 0 {
		b.Reset()
	}
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}
	if b.buf == nil && n <= smallBufferSize {
		b.buf = make([]T, n, smallBufferSize)
		return 0
	}
	c := cap(b.buf)
	if n <= c/2-m {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(b.buf, b.buf[b.off:])
	} else if c > maxInt-c-n {
		panic(ErrTooLarge)
	} else {
		// Add b.off to account for b.buf[:b.off] being sliced off the front.
		b.buf = growSlice(b.buf[b.off:], b.off+n)
	}
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}
