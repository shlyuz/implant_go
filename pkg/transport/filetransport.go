//go:build filetransport
// +build filetransport

package transport

type TransportInfo struct {
	Name        string
	Author      string
	Description string
	Version     string
}
type TransportComponent struct {
	Info TransportInfo
}
