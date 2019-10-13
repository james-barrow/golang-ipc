package ipc

import "errors"

//  returns the status of the connection as a string
func (status *Status) statusString() string {

	switch *status {
	case NotConnected:
		return "Not Connected"
	case Connected:
		return "Connected"

	case Listening:
		return "Listening"

	case ReConnecting:
		return "Re-connecting"
	case Closed:
		return "Closed"
	case Error:
		return "Error"
	default:
		return "Status not found"
	}
}

// checks the name passed into the start function to ensure it's ok/will work.
func checkIpcName(ipcName string) error {

	if len(ipcName) == 0 {
		return errors.New("ipcName cannot be an empty string")
	}

	return nil

}
