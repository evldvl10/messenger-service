package listener

import (
	"fmt"

	"messenger-service/event"
)

var (
	ApiChannel = make(chan event.EventChannelData)
)

func Api() {
	for event := range ApiChannel {
		fmt.Println(event)
	}
}
