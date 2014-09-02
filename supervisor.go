package supervisor

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/kolo/xmlrpc"
)

var supervisorURL, _ = url.Parse("http://localhost/RPC2")

type State struct {
	Statecode SupervisorState `xmlrpc:"statecode"`
	Statename SupervisorCode  `xmlrpc:"statename"`
}

type ProcessInfo struct {
	Name          string `xmlrpc:"name"`
	Group         string `xmlrpc:"group"`
	Start         int64  `xmlrpc:"start"`
	Stop          int64  `xmlrpc:"stop"`
	Now           int64  `xmlrpc:"now"`
	State         int64  `xmlrpc:"state"`
	Statename     string `xmlrpc:"statename"`
	StdoutLogfile string `xmlrpc:"stdout_logfile"`
	StderrLogfile string `xmlrpc:"stderr_logfile"`
	SpawnErr      string `xmlrpc:"spawnerr"`
	ExitStatus    int64  `xmlrpc:"exitstatus"`
	Pid           int64  `xmlrpc:"pid"`
}

type Supervisor interface {
	//status and control

	GetAPIVersion() (string, error)
	GetSupervisorVersion() (string, error)
	GetIdentification() (string, error)
	GetState() (State, error)
	GetPID() (int, error)
	ReadLog(offset, length int) (string, error)
	ClearLog() (bool, error)
	Shutdown() (bool, error)
	Restart() (bool, error)
	ReloadConfig() ([]string, []string, []string, error)

	//process control

	GetProcessInfo(string) (ProcessInfo, error)
	GetAllProcessInfo() ([]ProcessInfo, error)
	StartProcess(string, bool) (bool, error)
	StartAllProcesses(bool) ([]ProcessInfo, error)
	StartProcessGroup(string, bool) ([]ProcessInfo, error)
	StopProcess(string, bool) (bool, error)
	StopAllProcesses(bool) ([]ProcessInfo, error)
	StopProcessGroup(string, bool) ([]ProcessInfo, error)
	SendProcessStdin(string, string) (bool, error)
	SendRemoteCommEvent(string, string) (bool, error)
	AddProcessGroup(string) (bool, error)
	RemoveProcessGroup(string) (bool, error)

	//process logging

	ReadProcessStdoutLog(string, int64, int64) (string, error)
	ReadProcessStderrLog(string, int64, int64) (string, error)
	TailProcessStdoutLog(string, int64, int64) (string, int64, bool, error)
	TailProcessStderrLog(string, int64, int64) (string, int64, bool, error)
	ClearProcessLogs(string) (bool, error)
	ClearAllProcessLogs() (bool, error)

	// misc
	Close() error
}

type supervisor struct {
	rpcClient *xmlrpc.Client
}

func (s *supervisor) startStopProcess(action, name string, wait bool) (success bool, err error) {
	err = s.rpcClient.Call(fmt.Sprintf("supervisor.%sProcess", action), []interface{}{name, wait}, &success)
	return
}

func (s *supervisor) multiProcessAction(method string, args interface{}) (info []ProcessInfo, err error) {
	err = s.rpcClient.Call(fmt.Sprintf("supervisor.%s", method), args, &info)
	return
}

func (s *supervisor) readProcessLog(source, name string, offset, length int64) (result string, err error) {
	err = s.rpcClient.Call(fmt.Sprintf("supervisor.readProcessStd%sLog", source),
		[]interface{}{name, offset, length}, &result)
	return
}

func (s *supervisor) tailProcessLog(source, name string, inOffset, length int64) (result string, offset int64, overflow bool, err error) {
	var values []interface{}
	if err = s.rpcClient.Call(fmt.Sprintf("supervisor.tailProcessStd%sLog", source),
		[]interface{}{name, offset, length}, &values); err != nil {
		return
	}

	// values should contain [string bytes, int offset, bool overflow]
	if len(values) != 3 {
		err = errors.New("tailProcessLog: array length != 3")
		return
	}

	var ok bool

	if result, ok = values[0].(string); !ok {
		goto bad_type
	}

	if offset, ok = values[1].(int64); !ok {
		goto bad_type
	}

	if overflow, ok = values[2].(bool); !ok {
		goto bad_type
	}
	return

bad_type:
	err = errors.New("tailProcessLog: incompatible type in result array")
	return
}

func (s *supervisor) GetAPIVersion() (version string, err error) {
	err = s.rpcClient.Call("supervisor.getAPIVersion", nil, &version)
	return
}

func (s *supervisor) GetSupervisorVersion() (version string, err error) {
	err = s.rpcClient.Call("supervisor.getSupervisorVersion", nil, &version)
	return
}

func (s *supervisor) GetIdentification() (identification string, err error) {
	err = s.rpcClient.Call("supervisor.getIdentification", nil, &identification)
	return
}

func (s *supervisor) GetState() (state State, err error) {
	err = s.rpcClient.Call("supervisor.getState", nil, &state)
	return
}

func (s *supervisor) GetPID() (pid int, err error) {
	err = s.rpcClient.Call("supervisor.getPID", nil, &pid)
	return
}

func (s *supervisor) ReadLog(offset, length int) (log string, err error) {
	err = s.rpcClient.Call("supervisor.readLog", []interface{}{offset, length}, &log)
	return
}

func (s *supervisor) ClearLog() (success bool, err error) {
	err = s.rpcClient.Call("supervisor.clearLog", nil, &success)
	return
}

func (s *supervisor) Shutdown() (success bool, err error) {
	err = s.rpcClient.Call("supervisor.shutdown", nil, &success)
	return
}

func (s *supervisor) Restart() (success bool, err error) {
	err = s.rpcClient.Call("supervisor.restart", nil, &success)
	return
}

func (s *supervisor) ReloadConfig() (added, changed, removed []string, err error) {
	copyInterfaceToStringSlice := func(out []string, in interface{}) ([]string, error) {
		arr, ok := in.([]interface{})
		if !ok {
			return nil, errors.New("ReloadConfig: parameter not an array")
		}
		for _, s := range arr {
			if str, ok := s.(string); ok {
				out = append(out, str)
			} else {
				return nil, errors.New("ReloadConfig: array contains non-string")
			}
		}

		return out, nil
	}

	var status []interface{}

	//for some reason this returns [[added, changed, removed]]
	err = s.rpcClient.Call("supervisor.reloadConfig", nil, &status)
	if len(status) == 1 {
		if inner, ok := status[0].([]interface{}); ok && len(inner) == 3 {
			if added, err = copyInterfaceToStringSlice(added, inner[0]); err != nil {
				return
			}

			if changed, err = copyInterfaceToStringSlice(changed, inner[1]); err != nil {
				return
			}

			if removed, err = copyInterfaceToStringSlice(removed, inner[2]); err != nil {
				return
			}
		}

		//everything fine here
		return
	}

	err = errors.New("Unexpected data returned from supervisor.reloadConfig")
	return
}

func (s *supervisor) GetProcessInfo(name string) (info ProcessInfo, err error) {
	err = s.rpcClient.Call("supervisor.getProcessInfo", name, &info)
	return
}

func (s *supervisor) GetAllProcessInfo() ([]ProcessInfo, error) {
	return s.multiProcessAction("getAllProcessInfo", nil)
}

func (s *supervisor) StartProcess(name string, wait bool) (bool, error) {
	return s.startStopProcess("start", name, wait)
}

func (s *supervisor) StartAllProcesses(wait bool) ([]ProcessInfo, error) {
	return s.multiProcessAction("startAllProcesses", wait)
}

func (s *supervisor) StartProcessGroup(name string, wait bool) ([]ProcessInfo, error) {
	return s.multiProcessAction("startProcessGroup", []interface{}{name, wait})
}

func (s *supervisor) StopProcess(name string, wait bool) (bool, error) {
	return s.startStopProcess("stop", name, wait)
}

func (s *supervisor) StopAllProcesses(wait bool) ([]ProcessInfo, error) {
	return s.multiProcessAction("stopAllProcesses", wait)
}

func (s *supervisor) StopProcessGroup(name string, wait bool) ([]ProcessInfo, error) {
	return s.multiProcessAction("stopProcessGroup", []interface{}{name, wait})
}

func (s *supervisor) SendProcessStdin(name, chars string) (success bool, err error) {
	err = s.rpcClient.Call("supervisor.sendProcessStdin", []interface{}{name, chars}, &success)
	return
}

func (s *supervisor) SendRemoteCommEvent(eventType, data string) (success bool, err error) {
	err = s.rpcClient.Call("supervisor.sendRemoteCommEvent", []interface{}{eventType, data}, &success)
	return
}

func (s *supervisor) AddProcessGroup(name string) (success bool, err error) {
	err = s.rpcClient.Call("supervisor.addProcessGroup", name, &success)
	return
}

func (s *supervisor) RemoveProcessGroup(name string) (success bool, err error) {
	err = s.rpcClient.Call("supervisor.removeProcessGroup", name, &success)
	return
}

func (s *supervisor) ReadProcessStdoutLog(name string, offset, length int64) (string, error) {
	return s.readProcessLog("out", name, offset, length)
}

func (s *supervisor) ReadProcessStderrLog(name string, offset, length int64) (string, error) {
	return s.readProcessLog("err", name, offset, length)
}

func (s *supervisor) TailProcessStdoutLog(name string, offset, length int64) (string, int64, bool, error) {
	return s.tailProcessLog("out", name, offset, length)
}

func (s *supervisor) TailProcessStderrLog(name string, offset, length int64) (string, int64, bool, error) {
	return s.tailProcessLog("err", name, offset, length)
}

func (s *supervisor) ClearProcessLogs(name string) (success bool, err error) {
	err = s.rpcClient.Call("supervisor.clearProcessLogs", name, &success)
	return
}

func (s *supervisor) ClearAllProcessLogs() (success bool, err error) {
	err = s.rpcClient.Call("supervisor.clearAllProcessLogs", nil, &success)
	return
}

func (s *supervisor) Close() error {
	return s.rpcClient.Close()
}

type supervisorTransport struct {
}

func (st *supervisorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil {
		return nil, errors.New("unix: nil Request.URL")
	}

	if req.Header == nil {
		return nil, errors.New("unix: nil Request.Header")
	}

	if req.URL.Scheme != "unix" {
		panic("unix: unsupported protocol scheme")
	}

	sock, err := net.Dial("unix", req.URL.Path)
	if err != nil {
		return nil, err
	}
	defer sock.Close()

	//create shallow copy of request object
	newReq := new(http.Request)
	*newReq = *req

	newReq.URL = supervisorURL
	newReq.Write(sock)

	return http.ReadResponse(bufio.NewReader(sock), req)
}

// New returns a Supervisor interface type connected to the net.URL specified in u
//
// Optionally specify a http.Transport to use, will use default http.Transport if nil.
// This will also register a
func New(url string, transport *http.Transport) Supervisor {
	if transport == nil {
		transport = new(http.Transport)
	}

	//transport.RegisterProtocol("unix", new(supervisorTransport))

	//xmlrpc.NewClient never returns an error
	client, _ := xmlrpc.NewClient(url, transport)
	return &supervisor{client}
}
