package lsp

import (
	"encoding/json"
	"pkg.nimblebun.works/go-lsp"
)

type rpcCall struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      lsp.ID          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      lsp.ID          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    rpcErrorCode    `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type nothing struct{}

type traceNotificationParams struct {
	Value lsp.TraceType `json:"value"`
}

func (r rpcError) Error() string {
	return r.Message
}

type rpcErrorCode int

const (
	rpcParseError     rpcErrorCode = -32700
	rpcInvalidRequest rpcErrorCode = -32600
	rpcMethodNotFound rpcErrorCode = -32601
	rpcInvalidParams  rpcErrorCode = -32602
	rpcInternalError  rpcErrorCode = -32603

	rpcServerErrorStart      rpcErrorCode = -32099
	rpcServerNotInitialized  rpcErrorCode = -32002
	rpcUnknownErrorCode      rpcErrorCode = -32001
	rpcReservedErrorRangeEnd rpcErrorCode = 32000

	rpcLspReservedErrorRangeStart rpcErrorCode = -32899
	rpcRequestFailed              rpcErrorCode = -32803
	rpcServerCancelled            rpcErrorCode = -32802
	rpcContentModified            rpcErrorCode = -32801
	rpcRequestCancelled           rpcErrorCode = -32800
	rpcLspReservedErrorRangeEnd   rpcErrorCode = -32800
)

type rpcNotification struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
