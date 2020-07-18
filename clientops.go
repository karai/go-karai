package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

type clientHeader struct {
	ClientHeaderAppName    string `json:"client_header_app_name"`
	ClientHeaderAppVersion string `json:"client_header_app_version"`
	ClientHeaderPeerID     string `json:"client_header_peer_id"`
	ClientProtocolVersion  string `json:"client_protocol_version"`
}

func joinChannel(ktx, pubKey, signedKey, ktxCertFileName string, keyCollection *ED25519Keys) *websocket.Conn {
	if isCoordinator {
		fmt.Println(brightred + "This is for nodes running in client mode only.")
		return nil
	}
	fmt.Printf(brightcyan+"\nConnection request with ktx %s", ktx)

	// request a websocket connection
	var conn = requestSocket(ktx, "1")

	// using that connection, attempt to join the channel
	var joinedChannel = stateYourBusiness(conn, pubKey[:64])

	// parse channel messages
	socketMsgParser(ktx, pubKey, signedKey, joinedChannel, keyCollection)

	// return the connection
	return conn
}

func stateYourBusiness(conn *websocket.Conn, pubKey string) *websocket.Conn {

	// new users should send JOIN with the pubkey
	if isFNG {
		joinReq := "JOIN " + pubKey[:64]
		_ = conn.WriteMessage(1, []byte(joinReq))
	}
	// returning users should send RTRN and the signed CA cert
	if !isFNG {
		certString := readFile(selfCertFilePath)
		rtrnReq := "RTRN " + pubKey[:64] + " " + certString
		_ = conn.WriteMessage(1, []byte(rtrnReq))
	}
	return conn
}

func requestSocket(ktx, protocolVersion string) *websocket.Conn {
	urlConnection := url.URL{Scheme: "ws", Host: ktx, Path: "/api/v" + protocolVersion + "/channel"}
	conn, _, err := websocket.DefaultDialer.Dial(urlConnection.String(), nil)
	handle(brightred+"There was a problem connecting to the channel: "+brightcyan, err)
	return conn
}

func socketMsgParser(ktx, pubKey, signedKey string, conn *websocket.Conn, keyCollection *ED25519Keys) {
	_, joinResponse, err := conn.ReadMessage()
	handle("There was a problem reading the socket: ", err)
	if strings.HasPrefix(string(joinResponse), "WCBK") {
		fmt.Printf("\nConnected to %s", ktx)
		fmt.Printf("\nType \"send <JSON>\" to send a transaction.")
		isTrusted = true
		sendHandler(conn, keyCollection)
	}
	if strings.Contains(string(joinResponse), string(capkMsg)) {
		convertjoinResponseString := string(joinResponse)
		trimNewLinejoinResponse := strings.TrimRight(convertjoinResponseString, "\n")
		trimCmdPrefix := strings.TrimPrefix(trimNewLinejoinResponse, "CAPK ")
		ncasMsgtring := signKey(keyCollection, trimCmdPrefix[:64])
		composedNcasMsgtring := string(ncasMsg) + " " + ncasMsgtring
		_ = conn.WriteMessage(1, []byte(composedNcasMsgtring))
		_, certResponse, err := conn.ReadMessage()
		convertStringcertResponse := string(certResponse) // keys := generateKeys()
		trimNewLinecertResponse := strings.TrimRight(convertStringcertResponse, "\n")
		trimCmdPrefixcertResponse := strings.TrimPrefix(trimNewLinecertResponse, "CERT ")
		handle("There was an error receiving the certificate: ", err)
		ktxCertFileName := p2pConfigDir + "/" + ktx + ".cert"
		createFile(ktxCertFileName)
		writeFile(ktxCertFileName, trimCmdPrefixcertResponse[:192])
		fmt.Printf(brightgreen + "\nCert Name: ")
		fmt.Printf(white+"%s", ktxCertFileName)
		fmt.Printf(brightgreen + "\nCert Body: ")
		fmt.Printf(white+"%s", trimCmdPrefixcertResponse[:192])
		isFNG = false
	}
}

func sendV1Transaction(msg string, conn *websocket.Conn) {
	_ = conn.WriteMessage(1, []byte(msg))
}

// sendHandler This is a basic input loop that listens for
// a few words that correspond to functions in the app. When
// a command isn't understood, it displays the help menu and
// returns to listening to input.
func sendHandler(conn *websocket.Conn, keyCollection *ED25519Keys) {
	reader := bufio.NewReader(os.Stdin)
	for {
		// fmt.Printf("\n%v%v%v\n", white+"Type '", brightgreen+"menu", white+"' to view a list of trusted commands")
		fmt.Print(brightpurple + "TX> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if strings.Compare("help", text) == 0 {
			menu()
		} else if strings.Compare("?", text) == 0 {
			menu()
		} else if strings.Compare("send", text) == 0 {
			txBody := strings.TrimPrefix(text, "SEND ")
			fmt.Printf("client sending: %s", txBody)
			sendV1Transaction(txBody, conn)
		}
	}
}
