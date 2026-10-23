//proto/generate.go
package proto

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative event.proto
