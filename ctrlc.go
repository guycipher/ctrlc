/*
* ctrlc
* copyright (c) 2022 Alex Padula
* clipboard logger
 */

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"ctrlc/glfw"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	// EXTERN
	"github.com/gorilla/websocket"
)

// Structures

type Copied struct {
	CopiedAt string
	Content  string
}

// WS
var upgrader = websocket.Upgrader{} // use default options

// DES application key
var key string

var connections []Connection

type Connection struct {
	Conn *websocket.Conn
}

// main() - Application starts here
func main() {
	log.Print("CTRLC\n")

	// Init initializes the GLFW library.
	err := glfw.Init()
	if err != nil {
		panic(err)
	}

	// Check ENV variable for CTRLC AES Key, if not found application will generate and provide.
	if os.Getenv("CTRLC_AES_32") == "" {
		bytes := make([]byte, 32) //generate a random 32 byte key for AES-256
		if _, err := rand.Read(bytes); err != nil {
			log.Panicf("Error Could not generate AES: \n%s", err)
		}

		key = hex.EncodeToString(bytes)

		err = os.Setenv("CTRLC_AES_32", key)
		if err != nil {
			log.Printf("Error could not set environment variable: \n%s", err)
			os.Exit(1)
		}

		log.Printf("Key generated and environment variables set\n" +
			"REMEMBER YOUR KEY:\n" +
			key + "\n")
		log.Println("Starting..")
		time.Sleep(time.Second * 10)
	} else {
		key = os.Getenv("CTRLC_AES_32")
	}

	log.Println(os.Getenv("CTRLC_AES_32"))

	// Start ctrlc goroutine
	go ctrlC()

	// safely exit
	//runtime.Goexit()

	httpServer := http.Server{
		Addr: ":47222",
	}

	http.HandleFunc("/", httpApplication)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	http.HandleFunc("/ws", ws)

	log.Println("Locate UI here: http://localhost:47222")

	err = httpServer.ListenAndServe()
	if err != nil {
		log.Panicf("ERROR: %s", err)
		os.Exit(1)
	}

}

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	connections = append(connections, Connection{Conn: c})
}

// sendWSMessage(msg string) - Send msg to all clients
func sendWSMessage(msg Copied) {
	for _, c := range connections {
		err := c.Conn.WriteJSON(map[string]Copied{"newCopy": msg})
		if err != nil {
			log.Print(err)
		}
	}
}

func encryptAES(stringToEncrypt string) (encryptedString string) {

	//Since the key is in string, we need to convert decode it to bytes
	keyBytes, _ := hex.DecodeString(key)
	plaintext := []byte(stringToEncrypt)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		panic(err.Error())
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%x", ciphertext)
}

func decryptAES(encryptedString string) (decryptedString string) {

	keyBytes, _ := hex.DecodeString(key)
	enc, _ := hex.DecodeString(encryptedString)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		panic(err.Error())
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}

	return fmt.Sprintf("%s", plaintext)
}

// appendToDAT(copied, key string) - Append encrypted string to dat
func appendToDAT(copied string) {
	file, err := os.OpenFile("ctrlc.dat", os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Panicf("failed opening ctrlc file: %s", err)
	}

	copiedEncrypted := encryptAES(copied)
	ts := strconv.Itoa(int(time.Now().Unix()))

	file.WriteString(":<<" + ts + copiedEncrypted + "\r\n")
	if err != nil {
		log.Panicf("failed writing to ctrlc file: %s", err)
	}
}

// readCtrlC() - Read ctrlc dat file
func readCtrlC() []byte {
	data, err := ioutil.ReadFile("ctrlc.dat")
	if err != nil {
		log.Panicf("failed reading data from file: %s", err)
	}

	return data
}

// ctrlC(key string) - GO routine that checks contents of system clipboard, if different than last encrypt and append to ctrlc dat file
func ctrlC() {
	for range time.Tick(time.Second * 1) {

		log.Printf("Current clipboard log size: %d", len(strings.Split(string(readCtrlC()), ":<<"))-1)

		copied := glfw.GetClipboardString()

		if len(copied) >= 5 {
			lastCopiedFormat := strings.TrimRight(strings.TrimLeft(strings.Split(string(readCtrlC()), ":<<")[len(strings.Split(string(readCtrlC()), ":<<"))-1], ":<<"), "\r\n")

			if lastCopiedFormat != "" {

				// remove unix tz from last inserted
				lastCopied := lastCopiedFormat[10:len(lastCopiedFormat)]
				log.Println(lastCopied)

				// Check if last row in DAT is same as currently copied, IF NOT append
				if copied != decryptAES(lastCopied) {
					log.Println("Last copied is different than last inserted.")
					appendToDAT(copied)

					// get latest
					latest := decryptAndUnmarshal()
					sendWSMessage(latest[len(latest)-1])
				}
			} else {
				log.Println("Initial insert.")
				appendToDAT(copied)
			}
		} else {
			log.Printf("Copied string not long enough to append: only %d long", len(copied))
		}

	}
}

func decryptAndUnmarshal() []Copied {
	clipboardRaw := strings.Split(string(readCtrlC()), ":<<")
	var clipboard []Copied

	for _, copied := range clipboardRaw {
		log.Println("fuck")

		if copied != "" {
			copiedFormat := strings.TrimRight(strings.TrimLeft(copied, ":<<"), "\r\n")

			// remove unix tz from last inserted
			c := copiedFormat[10:]
			if len(c) > 4 {
				log.Println("rip timestamp")
				log.Println(copiedFormat[0:10])
				copied := Copied{
					CopiedAt: copiedFormat[0:10],
					Content:  decryptAES(c),
				}

				clipboard = append(clipboard, copied)
			}
		}

	}

	return clipboard
}

func httpApplication(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("ctrlc.html")
	d := decryptAndUnmarshal()

	err := t.Execute(w, d)
	if err != nil {
		return
	}
}
