// Package debug serves the loopback-only native gRPC debug API.
//
// It reads snapshots from ports.Engine and posts write events to ports.Inbox.
// Authentication and peer checks are installed on the native gRPC server.
package debug
