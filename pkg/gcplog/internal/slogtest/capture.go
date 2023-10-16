package slogtest

// Copyright 2023 Jussi Kalliokoski
//
// Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted, provided that the above copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

import (
	"encoding/json"
	"sync"
)

// Capture is an io.Writer that unmarshals the written data into entries of
// type T, to be later retrieved with Entries(). Written buffers must be valid
// JSON by themselves, and if the unmarshaling errors, Write will return an
// error.
type Capture[T any] struct {
	m       sync.Mutex
	entries []T
}

// Write implements io.Writer.
func (c *Capture[T]) Write(data []byte) (n int, err error) {
	n = len(data)

	var entry T
	if err = json.Unmarshal(data, &entry); err != nil {
		return n, err
	}

	c.m.Lock()
	defer c.m.Unlock()
	c.entries = append(c.entries, entry)

	return n, nil
}

// Entries returns the captured entries.
func (c *Capture[T]) Entries() []T {
	c.m.Lock()
	defer c.m.Unlock()
	return c.entries
}
