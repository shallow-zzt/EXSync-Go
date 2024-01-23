package server

import (
	"EXSync/core/internal/config"
	"EXSync/core/internal/modules/hashext"
	"EXSync/core/internal/server/client"
	"EXSync/core/internal/server/commands"
	"EXSync/core/internal/server/scan"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"runtime"
	"time"
)

//var socketManage map[string]any

type Server struct {
	scan.Scan
	StopNewConnections bool
	mergeSocketDict    map[string]map[string]net.Conn
}

func NewServer() *Server {
	// 设置使用线程
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 创建服务实例
	server := Server{
		mergeSocketDict: map[string]map[string]net.Conn{},
	}

	// 创建局域网扫描验证服务
	go func() {
		for {
			if server.StopNewConnections {
				return
			}
			server.ScanDevices()
			time.Sleep(10 * time.Second)
		}
	}()

	// 创建监听套接字
	port := config.Config.Server.Addr.Port
	go server.createDataSocket(port)
	go server.createCommandSocket(port + 1)

	return &server
}

// createCommandSocket 创建套接字对象
func (s *Server) createCommandSocket(port int) {
	address := fmt.Sprintf("%s:%d", config.Config.Server.Addr.IP, port)
	conn, err := net.Listen("tcp", address)
	if err != nil {
		return
	}
	for {
		if s.StopNewConnections {
			return
		}
		socket, err := conn.Accept()
		if err != nil {
			logrus.Debugf("address %s: %s", address, err)
			continue
		}
		go s.verifyCommandSocket(socket)
	}
}

// createDataSocket 创建套接字对象
func (s *Server) createDataSocket(port int) {
	address := fmt.Sprintf("%s:%d", config.Config.Server.Addr.IP, port)
	conn, err := net.Listen("tcp", address)
	if err != nil {
		return
	}
	for {
		if s.StopNewConnections {
			return
		}
		socket, err := conn.Accept()
		if err != nil {
			logrus.Debugf("address %s: %s", address, err)
			continue
		}
		go s.verifyDataSocket(socket)
	}
}

func (s *Server) verifyCommandSocket(commandSocket net.Conn) {
	defer func(commandSocket net.Conn) {
		err := commandSocket.Close()
		if err != nil {
			logrus.Warning(err)
		}
	}(commandSocket)
	addr := commandSocket.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	logrus.Infof("Starting to verify command socket connection from %s...", host)
	if hostInfo, ok := s.VerifyManage[host]; ok && hostInfo.AesKey != "" {
		// todo: 验证通过处理
		if dataSocket, ok := s.mergeSocketDict[host]["command"]; ok {
			go commands.NewCommandProcess(s.VerifyManage[host].AesKey, dataSocket, commandSocket)
			delete(s.mergeSocketDict, host)
		} else {
			s.mergeSocketDict[host] = map[string]net.Conn{
				"command": commandSocket,
			}
		}
	} else {

	}
}

func (s *Server) verifyDataSocket(dataSocket net.Conn) {
	defer func(dataSocket net.Conn) {
		err := dataSocket.Close()
		if err != nil {
			logrus.Warning(err)
		}
	}(dataSocket)
	addr := dataSocket.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	logrus.Infof("Starting to verify data socket connection from %s...", host)
	if hostInfo, ok := s.VerifyManage[host]; ok && hostInfo.AesKey != "" {
		// todo: 验证通过处理
		if commandSocket, ok := s.mergeSocketDict[host]["command"]; ok {
			go commands.NewCommandProcess(s.VerifyManage[host].AesKey, dataSocket, commandSocket)
			delete(s.mergeSocketDict, host)
		} else {
			s.mergeSocketDict[host] = map[string]net.Conn{
				"data": dataSocket,
			}
		}
	} else {

	}
}

func (s *Server) initClient(ip string) {
	clientMark := hashext.GetRandomStr(8)
	verifyInfo := s.VerifyManage[ip]
	aesKey := verifyInfo.AesKey
	remoteID := verifyInfo.RemoteID
	client := client.Client{
		CommandSocket:     nil,
		DataSocket:        nil,
		CommandSocketPort: 0,
		DataSocketPort:    0,
		HostInfo:          nil,
		IP:                "",
	}
}
