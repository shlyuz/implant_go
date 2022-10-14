package transport

// Realistically, on a production deployment, you should only support the transports you're using during the operation

import (
	"fmt"
	"log"

	"shlyuz/internal/debugLp/pkg/component"
	"shlyuz/internal/debugLp/pkg/transport/filetransport"
	"shlyuz/pkg/utils/idgen"
	"shlyuz/pkg/utils/logging"
)

type TransportMethod interface {
	Initalize(Component *component.Component) (bool, error)
	Send(Component *component.Component) (bool, error)
	Recv(Component *component.Component) ([]byte, bool, error)
}

var transportMethods map[string]func([]string) (TransportMethod, bool, error)

func NewError(text string) error {
	return &UnsupportedTransportError{text}
}

type UnsupportedTransportError struct {
	s string
}

func (e *UnsupportedTransportError) Error() string {
	return e.s
}

type UnsupportedTransportMethod struct {
	Method    string
	Arguments []string
}

func (t *UnsupportedTransportMethod) Initalize(Component *component.Component) (bool, error) {
	err := &UnsupportedTransportError{}
	return false, err
}

func (t *UnsupportedTransportMethod) Send(Component *component.Component) (bool, error) {
	return false, nil
}

func (t *UnsupportedTransportMethod) Recv(Component *component.Component) ([]byte, bool, error) {
	return nil, false, nil
}

func newUnsupportedTransportMethod(arguments []string) (TransportMethod, bool, error) {
	return &UnsupportedTransportMethod{"UNSUPPORTED", []string{}}, false, &UnsupportedTransportError{}
}

func (t *FileTransportMethod) Initalize(Component *component.Component) (bool, error) {
	// Do your initalization stuff here for your transport
	//   for example:
	// filetransport.Initalize(Component)
	return true, nil
}

func (t *FileTransportMethod) Send(Component *component.Component) (bool, error) {
	err := filetransport.Send(Component)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (t *FileTransportMethod) Recv(Component *component.Component) ([]byte, bool, error) {
	data, err := filetransport.Recv(Component)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// Add transport structs, Transport(), Send(), Recv(), and new*Transport functions here

type FileTransportMethod struct {
	Method       string
	TransportId  string
	TransportDir string
}

func newFileTransportMethod(arguments []string) (TransportMethod, bool, error) {
	return &FileTransportMethod{"file_transport", idgen.GenerateId(), "~/tmp/shlyuztest/"}, true, nil
}

func init() {
	transportMethods = make(map[string]func([]string) (TransportMethod, bool, error))
	// add new methods here
	transportMethods["UNSUPPORTED"] = newUnsupportedTransportMethod
	transportMethods["file_transport"] = newFileTransportMethod
	// methods["rev_tcp_socket"] = newRevTCPMethod
}

func getTransport(method string, arguments []string, Component *component.Component) (TransportMethod, bool, error) {
	factory, ok := transportMethods[method]
	if !ok {
		return nil, false, fmt.Errorf("transport method '%s' not found", method)
	}
	return factory(arguments)
}

func PrepareTransport(Component *component.Component, methodArgs []string) (TransportMethod, bool, error) {
	log.SetPrefix(logging.GetLogPrefix())
	transport, _, err := getTransport(Component.Config.TransportName, methodArgs, Component)
	if err != nil {
		log.Println("invalid arguments for PrepareTransport: ", err)
	}
	// boolSuccess, err := TransportMethod.Initalize(transport, Component)
	return transport, true, err
}
