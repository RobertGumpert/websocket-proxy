package wsserver

import "ws-server/src/model"

type retryCounter int

type retryMessageWrapper struct {
	Message        model.WSMessage
	ClientCounters map[model.RemoteAddress]retryCounter
}
