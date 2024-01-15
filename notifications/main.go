package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"strconv"
	"syscall"
	"time"

	chPkg "exp/notifications/channels"
)

// seed the random number generator
var (
	r        = rand.New(rand.NewSource(123))
	channels = chPkg.NewChannels(10)
	sidIndex int
)

const (
	SendSMSInterval = 100 * time.Millisecond
	SendSMSTimeout  = 300 * time.Millisecond
)

// sendSMS simulates the sending of an SMS.
func sendSMS(ctx context.Context) {
	// Let's simulate that we are sending an SMS and we receive a sid (uuid)
	time.Sleep(SendSMSInterval)
	sidIndex++
	sid := strconv.Itoa(sidIndex)
	// Create an entry in the map with the sid as key and nil as value
	channels.Add(sid)
	log.Printf("SMS sent with sid: %s", sid)
	go markSuccessOrRetry(ctx, sid)
}

// webHook simulates the receiving of a webhook.
// It will receive a webhook with a result for a certain sid
// and it will signal that the SMS was delivered.
// If the sid is not found in the map, it will ignore it.
func webHook(ctx context.Context) {
	var waitingSid string
	var uuids []string
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fetchedUUIDs := channels.Keys()
			// sort the uuids
			sort.Strings(fetchedUUIDs)
			// check if the uuids are different
			if len(fetchedUUIDs) == 0 || reflect.DeepEqual(uuids, fetchedUUIDs) {
				continue
			}
			uuids = fetchedUUIDs
			sid := uuids[r.Intn(len(uuids))]
			if waitingSid == sid {
				continue
			}
			waitingSid = sid
			rt := time.Duration(r.Intn(500)) * time.Millisecond
			go func(sid string, rt time.Duration) {
				// Let's simulate that we receive the response for certain sid
				log.Printf("\tWaiting to receive the response for SID: %s (%s)", sid, rt)
				time.Sleep(rt)
				// check if key exists and it's nil
				if channels.Exists(sid) && channels.Get(sid) == nil {
					// Signal that the SMS was delivered
					channels.Send(sid)
					log.Printf("\tSignaling the SMS with sid %s was delivered", sid)
					return
				}
				log.Printf("\ttoo late to signal that the SMS with sid %s was delivered", sid)
			}(sid, rt)
		}

	}

}

func markSuccessOrRetry(ctx context.Context, sid string) {
	// A ticker is used to trigger operations at regular intervals.
	ticker := time.NewTicker(SendSMSTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-channels.Get(sid):
			log.Printf("++ Got the signal that SMS with sid %s was delivered", sid)
			channels.Shutdown(sid)
			return
		case <-ctx.Done():
			log.Printf("++ shutting down... %s", sid)
			return
		case <-ticker.C:
			log.Printf("++ SMS with sid %s was not delivered, retrying...", sid)
			channels.Shutdown(sid)
			// Retry SMS
			sendSMS(ctx)
			return
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go webHook(ctx)

	for i := 0; i < 5; i++ {
		sendSMS(ctx)
	}

	time.Sleep(5 * time.Second)
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	<-ctx.Done()
	log.Println("Shutting down...")
	time.Sleep(100 * time.Millisecond)
}
