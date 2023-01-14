//go:build windows

// 生成 proto
//go:generate protoc --proto_path=. --go_out=paths=source_relative:../ ./*.proto

// 生成 proto grpc
//go:generate protoc --proto_path=. --go-grpc_out=paths=source_relative:../ ./*.proto

// 生成 proto http
//go:generate protoc --proto_path=. --go-http_out=paths=source_relative:../ ./*.proto

// 生成 proto errors
//go:generate protoc --proto_path=. --go-errors_out=paths=source_relative:../ ./*.proto

// 生成 swagger
//go:generate protoc --proto_path=. --openapiv2_out ../ --openapiv2_opt logtostderr=true --openapiv2_opt json_names_for_fields=true ./*.proto

package proto
