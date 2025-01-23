package ipc

import (
	"encoding/json"
	"net"

	"hypr-exiled/pkg/global"
)

func SendCommand(command string) (Response, error) {
	log := global.GetLogger()

	log.Debug("Attempting to connect to socket server", "path", socketPath)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Error("Failed to connect to socket server", err)
		return Response{}, err
	}
	defer conn.Close()

	log.Debug("Connected to socket server", "remote_addr", conn.RemoteAddr())

	req := Request{Command: command}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		log.Error("Failed to encode request", err)
		return Response{}, err
	}

	log.Info("Request sent successfully", "command", command)

	var resp Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		log.Error("Failed to decode response", err)
		return Response{}, err
	}

	log.Info("Response received", "status", resp.Status, "message", resp.Message)
	return resp, nil
}
