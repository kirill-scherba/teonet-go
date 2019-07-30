package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kirill-scherba/net-example-go/udp/trudp"
)

func main() {
	fmt.Println("UDP test application ver 1.0.0")

	var (
		rhost string
		rport int
		rchan int
		port  int

		// Logger parameters
		logLevel string

		// Integer parameters
		maxQueueSize  int
		sendSleepTime int

		// Control flags parameters
		noLogTime  bool
		sendTest   bool
		showStat   bool
		sendAnswer bool
	)

	flag.IntVar(&maxQueueSize, "Q", trudp.DefaultQueueSize, "maximum send and receive queues size")
	flag.BoolVar(&noLogTime, "no-log-time", false, "don't show time in application log")
	flag.IntVar(&port, "p", 0, "this host port (to remote hosts connect to this host)")
	flag.StringVar(&rhost, "a", "", "remote host address (to connect to remote host)")
	flag.IntVar(&rchan, "c", 1, "remote host channel (to connect to remote host)")
	flag.IntVar(&rport, "r", 0, "remote host port (to connect to remote host)")
	flag.StringVar(&logLevel, "log", "CONNECT", "application log level")
	flag.IntVar(&sendSleepTime, "t", 0, "send timeout in microseconds")
	flag.BoolVar(&sendTest, "send-test", false, "send test data")
	flag.BoolVar(&sendAnswer, "answer", false, "send answer")
	flag.BoolVar(&showStat, "S", false, "show statistic")

	flag.Parse()

	for reconnectF := false; ; {

		tru := trudp.Init(port)

		// Set log level
		tru.LogLevel(logLevel, !noLogTime, log.LstdFlags|log.Lmicroseconds)

		// Set 'show statictic' flag
		tru.ShowStatistic(showStat)

		// Set default queue size
		tru.SetDefaultQueueSize(maxQueueSize)

		// Connect to remote server flag and send data when connected
		if rport != 0 {
			go func() {
				// Try to connect to remote hosr every 5 seconds
				for {
					tcd := tru.ConnectChannel(rhost, rport, rchan)

					// Auto sender flag
					tcd.SendTestMsg(sendTest)

					// Sender
					num := 0
					for tru.Running() {
						if sendSleepTime > 0 {
							time.Sleep(time.Duration(sendSleepTime) * time.Microsecond)
						}
						data := []byte("Hello-" + strconv.Itoa(num) + "!")
						err := tcd.WriteTo(data)
						if err != nil {
							break
						}
						num++
					}

					tru.Log(trudp.CONNECT, "(main) channel "+tcd.MakeKey()+" sender stopped")
					if !tru.Running() {
						break
					}
					time.Sleep(5 * time.Second)
					tru.Log(trudp.CONNECT, "(main) reconnect")
				}

				tru.Log(trudp.CONNECT, "(main) sender worker stopped")
			}()
		}

		// Receiver
		go func() {
			defer tru.ChanEventClosed()
			for ev := range tru.ChanEvent() {
				switch ev.Event {

				case trudp.GOT_DATA:
					tru.Log(trudp.DEBUG, "(main) GOT_DATA: ", ev.Data, string(ev.Data), fmt.Sprintf("%.3f ms", ev.Tcd.TripTime()))
					if sendAnswer {
						ev.Tcd.WriteTo([]byte(string(ev.Data) + " - answer"))
					}

				// case trudp.SEND_DATA:
				// 	tru.Log(trudp.DEBUG, "(main) SEND_DATA:", ev.Data, string(ev.Data))

				case trudp.INITIALIZE:
					tru.Log(trudp.CONNECT, "(main) INITIALIZE, listen at:", string(ev.Data))

				case trudp.DESTROY:
					tru.Log(trudp.CONNECT, "(main) DESTROY", string(ev.Data))

				case trudp.CONNECTED:
					tru.Log(trudp.CONNECT, "(main) CONNECTED", string(ev.Data))

				case trudp.DISCONNECTED:
					tru.Log(trudp.CONNECT, "(main) DISCONNECTED", string(ev.Data))

				case trudp.RESET_LOCAL:
					tru.Log(trudp.DEBUG, "(main) RESET_LOCAL executed at channel:", ev.Tcd.MakeKey())

				case trudp.SEND_RESET:
					tru.Log(trudp.DEBUG, "(main) SEND_RESET to channel:", ev.Tcd.MakeKey())

				default:
					tru.Log(trudp.ERROR, "(main)")
				}
			}
		}()

		// Ctrl+C process
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				switch sig {
				case syscall.SIGINT:
					// reconnectF = true
					// tru.Close()
					// return

					var str string
					fmt.Print("\033[2K\033[0E" + "Press Q to exit or R to reconnect: ")
					fmt.Scanf("%s\n", &str)
					switch str {
					case "r", "R":
						reconnectF = true
						tru.Close()
						return
					case "q", "Q":
						reconnectF = false
						tru.Close()
					}
				case syscall.SIGCLD:
					fallthrough
				default:
					fmt.Printf("sig: %x\n", sig)
				}
			}
		}()

		// Run trudp and start listen
		tru.Run()

		if !reconnectF {
			fmt.Println("bay...")
			break
		}
		fmt.Println("reonnect...")
	}
}
