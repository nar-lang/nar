package lsp

import (
	"encoding/json"
	"fmt"
	"log"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/normalized"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/ast/typed"
	"nar-compiler/internal/pkg/common"
	"os"
	"pkg.nimblebun.works/go-lsp"
	"reflect"
	"strings"
)

type server struct {
	cacheDir string
	log      *common.LogWriter

	rootURI          lsp.DocumentURI
	trace            lsp.TraceType
	workspaceFolders []lsp.WorkspaceFolder
	initialized      bool
	responseChan     chan rpcResponse
	notificationChan chan rpcNotification
	inChan           chan []byte
	compileChan      chan lsp.DocumentURI

	openedDocuments       map[lsp.DocumentURI]*lsp.TextDocumentItem
	documentToPackageRoot map[lsp.DocumentURI]string
	packageRootToName     map[string]ast.PackageIdentifier
	loadedPackages        map[ast.PackageIdentifier]*ast.LoadedPackage
	parsedModules         map[ast.QualifiedIdentifier]*parsed.Module
	normalizedModules     map[ast.QualifiedIdentifier]*normalized.Module
	typedModules          map[ast.QualifiedIdentifier]*typed.Module
}

type LanguageServer interface {
	Close()
	GotMessage(msg []byte)
}

func NewServer(cacheDir string, writeResponse func([]byte)) LanguageServer {
	s := &server{
		inChan:                make(chan []byte, 16),
		responseChan:          make(chan rpcResponse, 16),
		notificationChan:      make(chan rpcNotification, 128),
		compileChan:           make(chan lsp.DocumentURI, 1024),
		log:                   &common.LogWriter{},
		cacheDir:              cacheDir,
		openedDocuments:       map[lsp.DocumentURI]*lsp.TextDocumentItem{},
		documentToPackageRoot: map[lsp.DocumentURI]string{},
		packageRootToName:     map[string]ast.PackageIdentifier{},
		loadedPackages:        map[ast.PackageIdentifier]*ast.LoadedPackage{},
		parsedModules:         map[ast.QualifiedIdentifier]*parsed.Module{},
		normalizedModules:     map[ast.QualifiedIdentifier]*normalized.Module{},
		typedModules:          map[ast.QualifiedIdentifier]*typed.Module{},
	}
	go s.sender(writeResponse)
	go s.receiver()
	go s.compiler()
	return s
}

func (s *server) Close() {
	close(s.responseChan)
	close(s.notificationChan)
	close(s.inChan)
	close(s.compileChan)
}

func (s *server) GotMessage(msg []byte) {
	s.inChan <- msg
}

func (s *server) receiver() {
	for {
		msg, ok := <-s.inChan
		if !ok {
			break
		}
		if err := s.handleMessage(msg); err != nil {
			log.Println(err.Error())
		}
	}
}

func (s *server) sender(writeResponse func([]byte)) {
	for {
		var data []byte
		var err error
		select {
		case response, ok := <-s.responseChan:
			if !ok {
				break
			}
			data, err = json.Marshal(response)
			break
		case notification, ok := <-s.notificationChan:
			if !ok {
				break
			}
			data, err = json.Marshal(notification)
			break
		}
		if err != nil {
			s.log.Err(err)
		} else {
			writeResponse(data)
		}

		s.log.Flush(os.Stdout)
	}
}

func (s *server) handleMessage(msg []byte) error {
	var call rpcCall
	if err := json.Unmarshal(msg, &call); nil != err {
		return err
	}

	println("<- " + call.Method)

	response := rpcResponse{
		Jsonrpc: "2.0",
		Id:      call.Id,
		Error:   nil,
		Result:  []byte("null"),
	}
	needResponse := true

	v := reflect.ValueOf(s)
	methodName := strings.ReplaceAll(call.Method, "$", "S")
	methodName = strings.ReplaceAll(methodName, "/", "_")
	methodName = strings.ToUpper(methodName[0:1]) + methodName[1:]
	fn := v.MethodByName(methodName)
	if fn.IsValid() {
		paramType := fn.Type().In(0).Elem()
		param := reflect.New(paramType)
		paramIface := param.Interface()
		if err := json.Unmarshal(call.Params, paramIface); nil != err {
			return err
		}
		results := fn.Call([]reflect.Value{param})

		if len(results) == 1 {
			needResponse = false
			err, _ := results[0].Interface().(error)
			if nil != err {
				return err
			}
		} else {
			err, _ := results[1].Interface().(error)

			if nil != err {
				return err
			}
			if result, err := json.Marshal(results[0].Interface()); err != nil {
				return err
			} else {
				response.Result = result
			}
		}
	} else {
		response.Error = &rpcError{
			Code:    rpcMethodNotFound,
			Message: fmt.Sprintf("Method %s not implemented", call.Method),
		}
	}
	if needResponse {
		s.responseChan <- response
	}
	return nil
}

func (s *server) notify(message string, params any) {
	println("-> " + message)

	if data, err := json.Marshal(params); err == nil {
		s.notificationChan <- rpcNotification{
			Jsonrpc: "2.0",
			Method:  message,
			Params:  data,
		}
	}
}
