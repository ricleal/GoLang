package channels_test

import (
	"fmt"

	chPkg "exp/notifications/channels"
)

func ExampleChannels() {

	channels := chPkg.NewChannels(1)

	uuid0 := "00000000-0000-0000-0000-000000000000"
	uuid1 := "00000000-0000-0000-0000-000000000001"

	channels.Add(uuid0)
	if channels.Exists(uuid0) {
		fmt.Println("Exists", uuid0)
	}
	if !channels.Exists(uuid1) {
		fmt.Println("Does not exist", uuid1)
	}

	ret := channels.Get(uuid0)
	if ret != nil {
		fmt.Println("Got non-nil")
	}

	channels.Send(uuid0)
	fmt.Println("Sent", uuid0)

	ret = channels.Get(uuid0)
	if ret != nil {
		fmt.Println("Got:", <-ret)
	}

	channels.Close(uuid0)

	channels.Remove(uuid0)
	if !channels.Exists(uuid0) {
		fmt.Println("Does not exist", uuid0)
	}

	ret = channels.Get(uuid0)
	if ret == nil {
		fmt.Println("Got nil")
	}

	channels.Add(uuid1)
	keys := channels.Keys()
	fmt.Println("Keys:", keys)
	channels.Shutdown(uuid1)
	ret = channels.Get(uuid1)
	if ret == nil {
		fmt.Println("Got nil")
	}

	fmt.Println("Done")
	// Output:
	// Exists 00000000-0000-0000-0000-000000000000
	// Does not exist 00000000-0000-0000-0000-000000000001
	// Sent 00000000-0000-0000-0000-000000000000
	// Got: {}
	// Does not exist 00000000-0000-0000-0000-000000000000
	// Got nil
	// Keys: [00000000-0000-0000-0000-000000000001]
	// Got nil
	// Done
}
