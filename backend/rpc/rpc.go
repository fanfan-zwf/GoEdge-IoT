package rpc

import (
	"main/Init"

	"log"
	"net"
	"net/rpc"
)

type FileService struct{}

// 定义 RPC 方法，注意参数是指针
func (f *FileService) Upload(data []byte, reply *bool) error {
	log.Printf("Received data size: %d bytes\n", len(data))
	// 这里可以保存文件
	*reply = true
	return nil
}

func New() {
	rpc.Register(new(FileService))
	listener, err := net.Listen(Init.Config.RPC.Network, Init.Config.RPC.Network)
	if err != nil {
		log.Fatal(err)
	}
	rpc.Accept(listener)
}
