package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const httpPort = ":7000"
const otherPort = ":7001"
const entryRelay = "EN"
const middleRelay = "I"
const exitRelay = "EX"

// Node is struct that stores the relay info
type Node struct {
	RelayType string
	IPandPort string
	PubKey    string
}

type torData struct {
	MidRelay  string
	ExitRelay string
	URL       string
	pageBody  string
}

var clientConnection net.Conn

func sendAliveMessages(conn net.Conn) {
	for {
		d, _ := time.ParseDuration("2s")
		time.Sleep(d)
		conn.Write([]byte("ALIVE\n"))
	}
}

func clientHandler(rw http.ResponseWriter, r *http.Request) {
	//Get list from directory server
	//Generate a random path of relays
	//Send to entry relay: address of mid and exit relay and URL
	//Wait for response
	clientConnection.Write([]byte("GET_LIST\n"))
	buffer := make([]byte, 0, 1024)
	n, _ := clientConnection.Read(buffer)
	if buffer == nil || len(buffer) == 0 {
		rw.Write([]byte("<html><h2>ERROR! Not enough relays in TOR Network</h2></html>"))
		return
	}
	relayList := make([]Node, 0, 10)
	json.Unmarshal(buffer[:n], &relayList)
	entryRelayList := make([]Node, 0, 10)
	middleRelayList := make([]Node, 0, 10)
	exitRelayList := make([]Node, 0, 10)
	for _, relay := range relayList {
		if relay.RelayType == entryRelay {
			entryRelayList = append(entryRelayList, relay)
		} else if relay.RelayType == middleRelay {
			middleRelayList = append(middleRelayList, relay)
		} else {
			exitRelayList = append(exitRelayList, relay)
		}
	}

	chEntryRelay := entryRelayList[rand.Intn(len(entryRelayList))]
	chMidRelay := middleRelayList[rand.Intn(len(middleRelayList))]
	chExitRelay := exitRelayList[rand.Intn(len(exitRelayList))]

	tData := torData{MidRelay: chMidRelay.IPandPort, ExitRelay: chExitRelay.IPandPort, URL: r.URL.Path, pageBody: ""}

	conn, _ := net.Dial("tcp", chEntryRelay.IPandPort)
	buffer, _ = json.Marshal(tData)
	conn.Write(buffer)
	newBuffer := make([]byte, 500, 1024)
	n, _ = conn.Read(newBuffer)
	newTData := torData{}
	json.Unmarshal(newBuffer[:n], &newTData)
	rw.Write([]byte(newTData.pageBody))
}

func listenAsARelay(relayType string) {

	line, err := net.Listen("tcp", ":"+string(rand.Intn(3000)+4000))
	for err != nil {
		line, err = net.Listen("tcp", ":"+string(rand.Intn(3000)+4000))
	}
	for {
		con, _ := line.Accept()
		go handleRelayConnection(con, relayType)
	}
}

func handleRelayConnection(conn net.Conn, relayType string) {
	//Forward to next relay the info
	//or get page from server if you are exit relay
	buffer := make([]byte, 500, 1024)
	n, _ := conn.Read(buffer)
	tData := torData{}
	json.Unmarshal(buffer[:n], &tData)
	n = 0
	if relayType == entryRelay {
		midRelayConn, _ := net.Dial("tcp", tData.MidRelay)
		tData.MidRelay = ""
		newBuffer, _ := json.Marshal(tData)
		midRelayConn.Write(newBuffer)
		buffer = make([]byte, 500, 1024)
		n, _ = midRelayConn.Read(buffer)
	} else if relayType == middleRelay {
		exitRelayConn, _ := net.Dial("tcp", tData.ExitRelay)
		tData.ExitRelay = ""
		newBuffer, _ := json.Marshal(tData)
		exitRelayConn.Write(newBuffer)
		buffer = make([]byte, 500, 1024)
		n, _ = exitRelayConn.Read(buffer)
	} else {
		response, _ := http.Get(tData.URL)
		responseBody, _ := ioutil.ReadAll(response.Body)
		response.Body.Close()
		tData.pageBody = string(responseBody)
		buffer, _ = json.Marshal(tData)
	}
	if n != 0 {
		conn.Write(buffer[:n])
	} else {
		conn.Write(buffer)
	}
	conn.Close()
}

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))
	fmt.Println("Do you want to participate as a relay?")
	//Take input
	consoleReader := bufio.NewReader(os.Stdin)
	input, _ := consoleReader.ReadString('\n')
	input = strings.TrimSpace(input)
	msg := ""
	if strings.ToLower(input) == "yes" || strings.ToLower(input) == "y" {
		fmt.Println("1")
		fmt.Print("1. Entry Relay\n2. Middle Relay\n3. Exit Relay\nEnter your choice: ")
		choice, _ := consoleReader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "1":
			msg = entryRelay
			break
		case "2":
			msg = middleRelay
			break
		case "3":
			msg = exitRelay
			break
		default:
			break
		}
	}
	clientConnection, _ = net.Dial("tcp", "localhost:6000")
	if msg != "" {
		fmt.Println("1")
		clientConnection.Write([]byte(msg + "\n"))
		clientConnection.Write([]byte("KEY\n"))
		go sendAliveMessages(clientConnection)
		go listenAsARelay(msg)
	} else {
		clientConnection.Write([]byte("N\n"))
	}
	http.HandleFunc("/fastor/", clientHandler)
	err := http.ListenAndServe(":"+string(rand.Intn(3000)+3000), nil)
	for err != nil {
		err = http.ListenAndServe(":"+string(rand.Intn(3000)+3000), nil)
	}

}
