package logs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"strings"
)

var (
	CPick      func(string) string
	filterKeys []string
)

type Logs struct {
	URL string
}

func NewLogs(URL string) (l Logs) {
	l = Logs{
		URL: URL,
	}

	return
}

func (log *Logs) Tail() {
	extractFilterKeys()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	c, _, err := websocket.DefaultDialer.Dial(log.URL, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		panic(err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer c.Close()
		defer close(done)

		CPick = colorPicker()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				fmt.Fprintln(os.Stderr, "read:", err)
				os.Exit(2)
				return
			}

			var f []interface{}
			err = json.Unmarshal(message, &f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
				if e, ok := err.(*json.SyntaxError); ok {
					fmt.Fprintf(os.Stderr, "Syntax error at byte offset %d\n", e.Offset)
				}
				continue
			}

			l := f[1].(map[string]interface{})

			m := NewMessage(l)
			m.Print()
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				fmt.Fprintln(os.Stderr, "write:", err)
				return
			}
		case <-interrupt:
			fmt.Fprintln(os.Stderr, "interrupt")
			// To cleanly close a connection, a client should send a close
			// frame and wait for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				fmt.Fprintln(os.Stderr, "write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			c.Close()
			return
		}
	}
}

func colorPicker() func(string) string {
	noColor := viper.GetBool("no-color")

	arrayColors := []func(...interface{}) string{
		color.New(color.FgRed).SprintFunc(),
		color.New(color.FgGreen).SprintFunc(),
		color.New(color.FgYellow).SprintFunc(),
		color.New(color.FgBlue).SprintFunc(),
		color.New(color.FgMagenta).SprintFunc(),
		color.New(color.FgCyan).SprintFunc(),
		color.New(color.FgWhite).SprintFunc(),
	}

	idents := map[string]string{}

	i := 0
	picker := func(ident string) string {
		if noColor {
			return ident
		}

		if _, ok := idents[ident]; ok {
			return idents[ident]
		}
		idents[ident] = arrayColors[i](ident)
		if i >= (len(arrayColors) - 1) {
			i = 0
		} else {
			i++
		}
		return idents[ident]
	}
	return picker
}

type Message struct {
	Payload   map[string]interface{}
	Json      string
	Timestamp string
	Ident     string
	Message   string
}

func NewMessage(payload map[string]interface{}) (m Message) {
	m = Message{
		Payload: payload,
	}
	m.parse()
	return
}

func (m *Message) setTimestamp() {
	localTime := viper.GetBool("local-time")

	var timestamp time.Time

	if m.Payload["timestamp"] != nil {
		switch m.Payload["timestamp"].(type) {
		case float64:
			timestamp, _ = time.Parse(time.RFC3339Nano, fmt.Sprintf("%f", m.Payload["timestamp"].(float64)))
		default:
			timestamp, _ = time.Parse(time.RFC3339Nano, m.Payload["timestamp"].(string))
		}
		// timestamp, _ = time.Parse(time.RFC3339Nano, m.Payload["timestamp"].(string))
	} else if m.Payload["created"] != nil {
		timestamp, _ = time.Parse(time.RFC3339Nano, m.Payload["created"].(string))
	} else {
		timestamp = time.Now()
	}

	if localTime {
		m.Timestamp = fmt.Sprintf("%s", timestamp.In(time.Now().Location()).Format("2006-01-02 15:04:05.000"))
	} else {
		m.Timestamp = fmt.Sprintf("%s", time.Now().UTC().Format("2006-01-02 15:04:05.000"))
	}
}

func contains(s []string, k string) bool {
	for _, key := range s {
		if key == k {
			return true
		}
	}
	return false
}

func sliceIsSubset(slice, subset []string) bool {
	for _, k := range subset {
		if !contains(slice, k) {
			return false
		}
	}
	return true
}

func extractFilterKeys() {
	filters := viper.GetStringSlice("filter")
	for _, filterLine := range filters {
		s := strings.Split(filterLine, "=")
		filterKeys = append(filterKeys, s[0])
	}
}

func (m *Message) setIdent() {
	configIdents := viper.GetStringSlice("ident")
	verbose := viper.GetBool("verbose")
	debug := viper.GetBool("debug")

	idents := [][]string{}

	for _, configIdent := range configIdents {
		configIdent = strings.Replace(configIdent, " ", "", -1)
		idents = append(idents, strings.Split(configIdent, ","))
	}

	idents = append(idents, []string{"container_name", "stack_name"})
	if verbose {
		idents = append(idents, []string{"container_name", "pod_name", "namespace_name"})
	}
	idents = append(idents, []string{"container_name", "namespace_name"})
	idents = append(idents, []string{"container_name", "ecs_cluster", "task_defition"})
	idents = append(idents, []string{"command", "image_name"})
	idents = append(idents, []string{"container_id"})
	idents = append(idents, []string{"host"})
	idents = append(idents, []string{"source"})

	keys := make([]string, 0, len(m.Payload))
	for k, v := range m.Payload {
		if value, ok := v.(string); ok {
			if value == "" {
				continue
			}
		}
		keys = append(keys, k)
	}

	ident := []string{}
	for _, profile := range idents {
		if sliceIsSubset(keys, profile) {
			if debug {
				fmt.Printf("Profile selected %v\n", profile)
			}
			for idx, field := range profile {
				if contains(filterKeys, field) {
					continue
				}
				if idx == 0 {
					ident = append(ident, CPick(m.Payload[field].(string)))
				} else {
					ident = append(ident, m.Payload[field].(string))
				}
			}
			break
		}
	}

	if len(ident) > 0 {
		m.Ident = fmt.Sprintf("%s", strings.Join(ident[:], " "))
	}
}

func (m *Message) setMessage() {
	if m.Payload["message"] != nil {
		m.Message = strings.TrimSpace(m.Payload["message"].(string))
	} else if m.Payload["MESSAGE"] != nil {
		m.Message = strings.TrimSpace(m.Payload["MESSAGE"].(string))
	} else if m.Payload["short_message"] != nil {
		m.Message = strings.TrimSpace(m.Payload["short_message"].(string))
	} else if m.Payload["SHORT_MESSAGE"] != nil {
		m.Message = strings.TrimSpace(m.Payload["SHORT_MESSAGE"].(string))
	} else {
		m.Message = ""
	}
	m.Message = strings.Replace(m.Message, "\r", "\n", -1)
}

func (m *Message) Print() {
	rawOutput := viper.GetBool("raw-output")
	debug := viper.GetBool("debug")

	line := make([]string, 0, 3)
	if !rawOutput && m.Message != "" {
		if m.Timestamp != "" {
			line = append(line, m.Timestamp)
		}
		if m.Ident != "" {
			line = append(line, m.Ident)
		}
		if m.Message != "" {
			line = append(line, m.Message)
		}
		fmt.Printf("%s\n", strings.Join(line[:], " "))
	}

	if rawOutput || debug {
		fmt.Printf("%s\n", m.Json)
	}
}

func (m *Message) parse() {
	rawOutput := viper.GetBool("raw-output")
	debug := viper.GetBool("debug")

	if rawOutput || debug {
		jsonString, err := json.Marshal(m.Payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding payload: %v\n", err)
			if e, ok := err.(*json.SyntaxError); ok {
				fmt.Fprintf(os.Stderr, "Syntax error at byte offset %d\n", e.Offset)
			}
			return
		}
		m.Json = string(jsonString)
	}

	if rawOutput {
		return
	}

	m.setTimestamp()
	m.setIdent()
	m.setMessage()
}
