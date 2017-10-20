package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
//logFile *os.File
)

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	printWelcome()

	args := os.Args
	argc := len(os.Args)
	if argc <= 2 {
		printHelp()
		os.Exit(0)
	}

	//TODO:support UDP protocol

	/*var logFileError error
	if argc > 5 && args[4] == "-log" {
		logPath := args[5] + "/" + time.Now().Format("2006_01_02_15_04_05") // "2006-01-02 15:04:05"
		logPath += args[1] + "-" + strings.Replace(args[2], ":", "_", -1) + "-" + args[3] + ".log"
		logPath = strings.Replace(logPath, `\`, "/", -1)
		logPath = strings.Replace(logPath, "//", "/", -1)
		logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if logFileError != nil {
			log.Fatalln("[x]", "log file path error.", logFileError.Error())
		}
		log.Println("[√]", "open test log file success. path:", logPath)
	}*/

	switch args[1] {
	case "-listen":
		if argc < 3 {
			log.Fatalln(`-listen need two arguments, like "ptt -listen 1997 2017".`)
		}
		port1 := checkPort(args[2])
		port2 := checkPort(args[3])
		log.Println("[√]", "start to listen port:", port1, "and port:", port2)
		port2port(port1, port2)
		break
	case "-tran":
		if argc < 3 {
			log.Fatalln(`-tran need two arguments, like "ptt -tran 1997 192.168.1.2:3389".`)
		}
		port := checkPort(args[2])
		var remoteAddress string
		if checkIp(args[3]) {
			remoteAddress = args[3]
		}
		split := strings.SplitN(remoteAddress, ":", 2)
		log.Println("[√]", "start to transmit address:", remoteAddress, "to address:", split[0]+":"+port)
		port2host(port, remoteAddress)
		break
	case "-slave":
		if argc < 3 {
			log.Fatalln(`-slave need two arguments, like "ptt -slave 127.0.0.1:3389 8.8.8.8:1997".`)
		}
		var address1, address2 string
		checkIp(args[2])
		if checkIp(args[2]) {
			address1 = args[2]
		}
		checkIp(args[3])
		if checkIp(args[3]) {
			address2 = args[3]
		}
		log.Println("[√]", "start to connect address:", address1, "and address:", address2)
		host2host(address1, address2)
		break
	default:
		printHelp()
	}
}

func printWelcome() {
	fmt.Println("+----------------------------------------------------------+")
	fmt.Println("| Welcome to use the port transmit tool. (PTT ver1.0)      |")
	fmt.Println("| Code by cw1997 at 2017-10-19 03:59:51                    |")
	fmt.Println("| If you have some problem when you use the tool,          |")
	fmt.Println("| please submit issue at : https://github.com/cw1997/ptt . |")
	fmt.Println("+----------------------------------------------------------+")
	fmt.Println()
	// sleep one second because the fmt is not thread-safety.
	// if not to do this, fmt.Print will print after the log.Print.
	time.Sleep(time.Second)
}
func printHelp() {
	fmt.Println(`usage: "-listen port1 port2" example: "ptt -listen 1997 2017" `)
	fmt.Println(`       "-tran port1 ip:port2" example: "ptt -tran 1997 192.168.1.2:3389" `)
	fmt.Println(`       "-slave ip1:port1 ip2:port2" example: "ptt -slave 127.0.0.1:3389 8.8.8.8:1997" `)
	fmt.Println(`============================================================`)
	fmt.Println(`optional argument: "-log logpath" . example: "ptt -listen 1997 2017 -log d:/ptt" `)
	fmt.Println(`log filename format: Y_m_d_H_i_s-agrs1-args2-args3.log`)
	fmt.Println(`============================================================`)
	fmt.Println(`if you want more help, please read "README.md". `)
}

func checkPort(port string) string {
	PortNum, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalln("[x]", "port should be a number")
	}
	if PortNum < 1 && PortNum > 65535 {
		log.Fatalln("[x]", "port should be a number and the range is [1,65536)")
	}
	return port
}

func checkIp(address string) bool {
	pattern := `(\d|[1-9]\d|1\d{2}|2[0-5][0-5])\.(\d|[1-9]\d|1\d{2}|2[0-5][0-5])\.(\d|[1-9]\d|1\d{2}|2[0-5][0-5])\.(\d|[1-9]\d|1\d{2}|2[0-5][0-5]):([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5])`
	ok, err := regexp.MatchString(pattern, address)
	if err != nil || !ok {
		log.Fatalln("[x]", "ip address error. should be a string like [ip:port]. ")
	}
	return ok
}

func port2port(port1 string, port2 string) {
	listen1 := start_server("0.0.0.0:" + port1)
	listen2 := start_server("0.0.0.0:" + port2)
	log.Println("[√]", "listen port:", port1, "and", port2, "success. waiting for client...")
	for {
		conn1 := accept(listen1)
		if conn1 == nil {
			continue
		}
		conn2 := accept(listen2)
		if conn2 == nil {
			conn1.Close()
			continue
		}
		if conn1 != nil && conn2 != nil {
			forward(conn1, conn2)
		}
	}
}

func port2host(allowPort string, targetAddress string) {
	server := start_server("0.0.0.0:" + allowPort)
	for {
		conn := accept(server)
		if conn == nil {
			continue
		}
		//println(targetAddress)
		go func(targetAddress string) {
			target, err := net.Dial("tcp", targetAddress)
			if err != nil {
				// temporarily unavailable, don't use fatal.
				log.Println("[√]", "connect target address ["+targetAddress+"] faild.")
				return
			}
			log.Println("[→]", "connect target address ["+targetAddress+"] success.")
			forward(target, conn)
		}(targetAddress)
	}
}

func host2host(address1, address2 string) {
	for {
		host1, err := net.Dial("tcp", address1)
		if err != nil {
			log.Fatalln("[x]", "connect user's host ["+address1+"] faild.")
		}
		host2, err := net.Dial("tcp", address2)
		if err != nil {
			log.Fatalln("[x]", "connect target address ["+address2+"] faild.")
		}
		log.Println("[→]", "connect user's host ["+address1+"] and target address ["+address2+"] success.")
		forward(host1, host2)
	}
}

func start_server(address string) net.Listener {
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("[x]", "listen address ["+address+"] faild.")
	}
	log.Println("[√]", "start listen at address:["+address+"]")
	return server
	/*defer server.Close()

	for {
		conn, err := server.Accept()
		log.Println("accept a new client. remote address:[" + conn.RemoteAddr().String() +
			"], local address:[" + conn.LocalAddr().String() + "]")
		if err != nil {
			log.Println("accept a new client faild.", err.Error())
			continue
		}
		//go recvConnMsg(conn)
	}*/
}

func accept(listener net.Listener) net.Conn {
	conn, err := listener.Accept()
	if err != nil {
		log.Println("[x]", "accept connect ["+conn.RemoteAddr().String()+"] faild.", err.Error())
		return nil
	}
	log.Println("[√]", "accept a new client. remote address:["+conn.RemoteAddr().String()+"], local address:["+conn.LocalAddr().String()+"]")
	return conn
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	var wg sync.WaitGroup
	// wait tow goroutines
	wg.Add(4)
	go connCopy(conn1, conn2, &wg)
	go connCopy(conn2, conn1, &wg)
	//blocking when the wg is locked
	wg.Wait()
	conn1.Close()
	conn2.Close()
}

func connCopy(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup) {
	//TODO:log, record the data from conn1 and conn2.
	/*logFile := openLog(conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	if logFile != nil {
		w := io.MultiWriter(conn1, logFile)
		io.Copy(w, conn2)
	} else {
		io.Copy(conn1, conn2)
	}
	conn2.Close()
	log.Println("[←]", "close the connect at local:["+conn1.LocalAddr().String()+"] and remote:["+conn1.RemoteAddr().String()+"]")*/
	c := make(chan []byte, 0)
	go func(c chan []byte) {
		for buf := range c {
			n, err := conn2.Write(buf)
			if err != nil {
				log.Println("conn2.Close", n)
				break
			}
			log.Println("conn2.Write", n)
		}
		wg.Done()
	}(c)
	go func(c chan []byte) {
		for {
			buf := make([]byte, 1024)
			b := bytes.NewBuffer(buf)
			io.Copy(b, conn1)
			c <- buf
			/*//n, _ := conn1.Read(buf)
			if n != 0 {
				c <- buf
				log.Println("conn1.Read", n)
			}*/
			//log.Println("conn1.Read error", n)
		}
		wg.Done()
	}(c)
}
func openLog(address1, address2, address3, address4 string) *os.File {
	args := os.Args
	argc := len(os.Args)
	var logFileError error
	var logFile *os.File
	if argc > 5 && args[4] == "-log" {
		address1 = strings.Replace(address1, ":", "_", -1)
		address2 = strings.Replace(address2, ":", "_", -1)
		address3 = strings.Replace(address3, ":", "_", -1)
		address4 = strings.Replace(address4, ":", "_", -1)
		timeStr := time.Now().Format("2006_01_02_15_04_05") // "2006-01-02 15:04:05"
		logPath := args[5] + "/" + timeStr + args[1] + "-" + address1 + "_" + address2 + "-" + address3 + "_" + address4 + ".log"
		logPath = strings.Replace(logPath, `\`, "/", -1)
		logPath = strings.Replace(logPath, "//", "/", -1)
		logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if logFileError != nil {
			log.Fatalln("[x]", "log file path error.", logFileError.Error())
		}
		log.Println("[√]", "open test log file success. path:", logPath)
	}
	return logFile
}
