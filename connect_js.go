//go:build js

package ipc

// js/wasm stand-ins: there are no unix sockets or named pipes in a wasm
// runtime, so IPC connections fail cleanly. In-process consumers talk to
// their peer directly and never reach these paths.

import "errors"

var errIPCUnsupportedJS = errors.New("golang-ipc: no IPC transport on js/wasm")

func (s *Server) run() error { return errIPCUnsupportedJS }

func (c *Client) dial() error { return errIPCUnsupportedJS }
