package main

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"
	"strings"
)

type TerminalModes map[uint8]uint32

const (
	VINTR         = 1
	VQUIT         = 2
	VERASE        = 3
	VKILL         = 4
	VEOF          = 5
	VEOL          = 6
	VEOL2         = 7
	VSTART        = 8
	VSTOP         = 9
	VSUSP         = 10
	VDSUSP        = 11
	VREPRINT      = 12
	VWERASE       = 13
	VLNEXT        = 14
	VFLUSH        = 15
	VSWTCH        = 16
	VSTATUS       = 17
	VDISCARD      = 18
	IGNPAR        = 30
	PARMRK        = 31
	INPCK         = 32
	ISTRIP        = 33
	INLCR         = 34
	IGNCR         = 35
	ICRNL         = 36
	IUCLC         = 37
	IXON          = 38
	IXANY         = 39
	IXOFF         = 40
	IMAXBEL       = 41
	ISIG          = 50
	ICANON        = 51
	XCASE         = 52
	ECHO          = 53
	ECHOE         = 54
	ECHOK         = 55
	ECHONL        = 56
	NOFLSH        = 57
	TOSTOP        = 58
	IEXTEN        = 59
	ECHOCTL       = 60
	ECHOKE        = 61
	PENDIN        = 62
	OPOST         = 70
	OLCUC         = 71
	ONLCR         = 72
	OCRNL         = 73
	ONOCR         = 74
	ONLRET        = 75
	CS7           = 90
	CS8           = 91
	PARENB        = 92
	PARODD        = 93
	TTY_OP_ISPEED = 128
	TTY_OP_OSPEED = 129
)

var defaultCommands = []string{
	"cat /etc/release",
	"showrev -p",
}

var exitString = "q"

var username = "user"
var passwd = "pass"

var programName = "SISCommander"
var printingChannel chan string
var commands, servers []string
var commander = &Commander{}

type Commander struct{
    listeners []chan string
}

func (p *Commander) add(c chan string){
    p.listeners = append(p.listeners, c)
}

func (p *Commander) publish(command string){
    for _, c := range p.listeners{
        c <- command
    }
}

func sshAndPrint(server string, config *ssh.ClientConfig, commandChannel chan string) {

	for {
		command := <-commandChannel
		output := fmt.Sprintf(server + " ## " + command + " ## " + "\n")
		output = output + strings.Repeat("=",len(output)) + "\n"

		client, err := ssh.Dial("tcp", server, config)
		if err != nil {
			output = output + fmt.Sprintf("Failed to dial: "+err.Error()+"\n")
		}

		// Each ClientConn can support multiple interactive sessions,
		// represented by a Session.
		defer client.Close()
		// Create a session
		session, err := client.NewSession()
		if err != nil {
			output = output + fmt.Sprintf("unable to create session: %s\n", err)
		}
		defer session.Close()
		// Set up terminal modes
		modes := ssh.TerminalModes{
			ECHO:          0,     // disable echoing
			TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		// Request pseudo terminal
		if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
			output = output + fmt.Sprintf("request for pseudo terminal failed: %s\n", err)
		}

		var b bytes.Buffer
		session.Stdout = &b
		if err := session.Run(command); err != nil {
			output = output + fmt.Sprintf("Failed to run: "+err.Error()+"\n")
		}
		output = output + fmt.Sprintf(b.String()+"\n")
		printingChannel <- output
		time.Sleep(time.Nanosecond * 1)
	}
}

func Usage() {
	fmt.Printf("Usage: %s -i <servers input file> ( -h for help )\n", programName)
}

func publish(command string) { //publishes to all commandchannels spawned

	for _, c := range commander.listeners {
		c <- command
	}

}

func printRoutine() {

	for {
		msg := <-printingChannel
		fmt.Println(msg)
		time.Sleep(time.Nanosecond * 1)
	}

}

func fileOutRoutine() {

	t := time.Now()

	f, err := os.Create("output_" + strconv.Itoa(t.Year())+"_"+strconv.Itoa(int(t.Month()))+"_"+strconv.Itoa(t.Day())+"_"+strconv.Itoa(t.Hour())+"_"+strconv.Itoa(t.Minute()) +".txt")
	if err != nil {

	}
	defer f.Close()

	for {
		msg := <-printingChannel
		_, err := f.WriteString(msg)
		if err != nil {
			log.Print(err)
		}
		f.Sync()
		time.Sleep(time.Nanosecond * 1)
	}
}

func printAvailableCommands() {

	for i, value := range defaultCommands {
		fmt.Println(i, ") ", value)
	}
	fmt.Println("\n")
}

func awaitUserInput() {

	for {
		fmt.Println("Choose the command you want to execute ( ", exitString, " for exit) :\n")
		fmt.Println("----------------------------------------\n")

		printAvailableCommands()

		var input string
		fmt.Scanln(&input)
		index, err := strconv.Atoi(input)
		fmt.Println("\n")
		if (err == nil) && index < len(defaultCommands) {
			commander.publish(defaultCommands[index])
		} else {
			if input == exitString {
				os.Exit(0)
			} else {
				fmt.Println("!Please provide a valid command number!\n")
			}
		}
	}
}

func main() {

	//Print Logo...
	fmt.Println("##########################\n\tSISCommander\n#\t version 0.1    #\n##########################\n")

	printingChannel = make(chan string)

	var infile, commandsfile string

	var help = flag.Bool("h", false, "help")
	flag.StringVar(&infile, "i", "", "Input file csv format")
	flag.StringVar(&commandsfile, "c", "", "Commands file csv format")
	flag.Parse()

	if *help {
		Usage()
		os.Exit(0)
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
	}

	if infile != "" {

		fmt.Println("Reading Servers file. Please Wait.. \n")

		file, _ := os.Open(infile)

		defer file.Close()
		//
		reader := csv.NewReader(file)

		// options are available at:
		// http://golang.org/src/pkg/encoding/csv/reader.go?s=3213:3671#L94
		reader.Comma = ','
		lineCount := 0
		for {
			// read just one record, but we could ReadAll() as well
			record, err := reader.Read()
			// end-of-file is fitted into err
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("Error:", err)
				break
			}

			var server string
			server = record[0]
			server = server + ":22"

			c := make(chan string)
			commander.listeners = append(commander.listeners, c)

			go sshAndPrint(server, config, c)

			lineCount += 1
		}
	} else {
		panic("No input file...")
	}

	if commandsfile != "" {

		fmt.Println("Reading Commands file. Please Wait.. \n")

		file, _ := os.Open(commandsfile)

		defer file.Close()
		//
		reader := csv.NewReader(file)

		reader.Comma = ','
		lineCount := 0
		for {
			// read just one record, but we could ReadAll() as well
			record, err := reader.Read()
			// end-of-file is fitted into err
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("Error:", err)
				break
			}

			commands = append(commands, record[0])
			commander.publish(record[0])

			lineCount += 1
		}
	}

	go fileOutRoutine()

	awaitUserInput()
}
